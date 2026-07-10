using System.Diagnostics;
using System.Globalization;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

using devo.config;
using devo.exceptions;
using devo.models.config;

namespace devo.services;

public sealed partial class AdoClient : IAdoClient
{
    private const string AdoResource = "499b84ac-1321-427f-aa17-267ca6975798";

    private readonly HttpClient _http;
    private readonly string _baseUrl;
    private readonly string _orgUrl;
    private readonly string _project;
    private readonly AuthMethod _authMethod;

    private readonly SemaphoreSlim _tokenLock = new(1, 1);
    private string? _token;
    private DateTimeOffset _tokenExp;

    public AdoClient(Config config, HttpClient http)
    {
        _http = http;
        _project = config.Project;
        _authMethod = config.AuthMethod;
        (_orgUrl, _baseUrl) = BuildUrls(config);

        if (config.AuthMethod == AuthMethod.Pat)
        {
            _token = config.Pat;
            _tokenExp = DateTimeOffset.MaxValue;
        }
    }

    private static (string _orgUrl, string _baseUrl) BuildUrls(Config config)
    {
        if (string.IsNullOrEmpty(config.OrgUrl))
        {
            // dev.azure.com default
            return ($"https://dev.azure.com/{config.Org}/_apis",
                    $"https://dev.azure.com/{config.Org}/{config.Project}/_apis");
        }

        string root = config.OrgUrl.TrimEnd('/');

        // strip project suffix if OrgUrl included it,
        // e.g. https://pdidev.visualstudio.com/PDI -> https://pdidev.visualstudio.com
        string projectSuffix = "/" + config.Project;
        if (root.EndsWith(projectSuffix, StringComparison.OrdinalIgnoreCase))
        {
            root = root[..^projectSuffix.Length];
        }

        return ($"{root}/_apis", $"{root}/{config.Project}/_apis");
    }

    // --- Auth ---

    private const string DefaultApiVersion = "7.1";
    private const string JsonContentType = "application/json";

    private string AuthHeader() =>
        "Basic " + Convert.ToBase64String(Encoding.UTF8.GetBytes(":" + _token));

    private async Task EnsureTokenAsync(CancellationToken ct)
    {
        if (_authMethod == AuthMethod.Pat)
        {
            return;
        }

        await _tokenLock.WaitAsync(ct);
        try
        {
            // refresh if token expires within 5 minutes
            if (_tokenExp - DateTimeOffset.Now < TimeSpan.FromMinutes(5))
            {
                await RefreshAzCliTokenAsync(ct);
            }
        }
        finally
        {
            _tokenLock.Release();
        }
    }

    private async Task ForceRefreshTokenAsync(CancellationToken ct)
    {
        await _tokenLock.WaitAsync(ct);
        try
        {
            await RefreshAzCliTokenAsync(ct);
        }
        finally
        {
            _tokenLock.Release();
        }
    }

    // caller must hold _tokenLock
    private async Task RefreshAzCliTokenAsync(CancellationToken ct)
    {
        var psi = new ProcessStartInfo("az")
        {
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
        };
        psi.ArgumentList.Add("account");
        psi.ArgumentList.Add("get-access-token");
        psi.ArgumentList.Add("--resource");
        psi.ArgumentList.Add(AdoResource);
        psi.ArgumentList.Add("--output");
        psi.ArgumentList.Add("json");

        using Process proc = Process.Start(psi)
            ?? throw new InvalidOperationException("failed to start az cli");
        string stdout = await proc.StandardOutput.ReadToEndAsync(ct);
        await proc.WaitForExitAsync(ct);
        if (proc.ExitCode != 0)
        {
            throw new InvalidOperationException(
                "az account get-access-token failed (is az cli logged in?)");
        }

        AzTokenResponse token = JsonSerializer.Deserialize<AzTokenResponse>(stdout)
            ?? throw new InvalidOperationException("failed to parse az token response");

        _token = token.AccessToken;

        // az cli returns local time like "2024-01-15 12:00:00.000000";
        // if unparseable, assume ~60 min tokens and refresh in 50
        _tokenExp = DateTime.TryParseExact(token.ExpiresOn, "yyyy-MM-dd HH:mm:ss.ffffff",
                CultureInfo.InvariantCulture, DateTimeStyles.None, out DateTime exp)
            ? new DateTimeOffset(exp)
            : DateTimeOffset.Now.AddMinutes(50);
    }

    private sealed record AzTokenResponse
    {
        [JsonPropertyName("accessToken")]
        public required string AccessToken { get; init; }

        [JsonPropertyName("expiresOn")]
        public string? ExpiresOn { get; init; }
    }

    // --- Request plumbing ---

    /// <summary>Executes a request with auth, api-version, and one 401 retry
    /// (az cli only). Returns the raw response body; throws ApiException on
    /// non-2xx.</summary>
    private async Task<string> ExecuteAsync(
        HttpMethod method,
        string url,
        object? body,
        string apiVersion,
        string contentType,
        string accept,
        CancellationToken ct)
    {
        await EnsureTokenAsync(ct);

        string fullUrl = url + (url.Contains('?') ? '&' : '?') + "api-version=" + apiVersion;
        string? json = body is null ? null : JsonSerializer.Serialize(body);

        HttpResponseMessage res = await SendOnceAsync();

        if (res.StatusCode == HttpStatusCode.Unauthorized)
        {
            res.Dispose();
            if (_authMethod != AuthMethod.AzCli)
            {
                throw new ApiException(401,
                    "authentication failed — check that your PAT is valid and has not expired");
            }
            await ForceRefreshTokenAsync(ct);
            res = await SendOnceAsync();
        }

        using (res)
        {
            string content = await res.Content.ReadAsStringAsync(ct);
            return !res.IsSuccessStatusCode ? throw new ApiException((int)res.StatusCode, content) : content;
        }

        async Task<HttpResponseMessage> SendOnceAsync()
        {
            using var req = new HttpRequestMessage(method, fullUrl);
            req.Headers.TryAddWithoutValidation("Authorization", AuthHeader());
            req.Headers.Accept.ParseAdd(accept);
            if (json is not null)
            {
                req.Content = new StringContent(json, Encoding.UTF8, contentType);
            }
            return await _http.SendAsync(req, ct);
        }
    }

    private async Task<T> SendAsync<T>(
        HttpMethod method,
        string url,
        object? body,
        string apiVersion = DefaultApiVersion,
        CancellationToken ct = default)
    {
        string content = await ExecuteAsync(
            method, url, body, apiVersion, JsonContentType, JsonContentType, ct);
        return JsonSerializer.Deserialize<T>(content)
            ?? throw new ApiException(0, "empty response body");
    }

    private Task<T> GetAsync<T>(string path, CancellationToken ct) =>
        SendAsync<T>(HttpMethod.Get, _baseUrl + path, body: null, ct: ct);

    private Task<T> GetOrgAsync<T>(string path, CancellationToken ct) =>
        SendAsync<T>(HttpMethod.Get, _orgUrl + path, body: null, ct: ct);

    private Task<T> GetPreviewAsync<T>(string path, string apiVersion, CancellationToken ct) =>
        SendAsync<T>(HttpMethod.Get, _baseUrl + path, body: null, apiVersion, ct);

    private Task<T> PostAsync<T>(string path, object body, CancellationToken ct) =>
        SendAsync<T>(HttpMethod.Post, _baseUrl + path, body, ct: ct);

    private Task PostAsync(string path, object body, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Post, _baseUrl + path, body,
            DefaultApiVersion, JsonContentType, JsonContentType, ct);

    private Task PostPreviewAsync(string path, string apiVersion, object body, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Post, _baseUrl + path, body,
            apiVersion, JsonContentType, JsonContentType, ct);

    private Task PutAsync(string path, object body, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Put, _baseUrl + path, body,
            DefaultApiVersion, JsonContentType, JsonContentType, ct);

    private Task PatchAsync(string path, object body, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Patch, _baseUrl + path, body,
            DefaultApiVersion, JsonContentType, JsonContentType, ct);

    /// <summary>PATCH with Content-Type application/json-patch+json — work item updates.</summary>
    private Task PatchJsonPatchAsync(string path, object body, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Patch, _baseUrl + path, body,
            DefaultApiVersion, "application/json-patch+json", JsonContentType, ct);

    /// <summary>Fetches raw (non-JSON) content — git item endpoints with $format=text.</summary>
    private Task<string> GetContentAsync(string path, CancellationToken ct) =>
        ExecuteAsync(HttpMethod.Get, _baseUrl + path, body: null,
            DefaultApiVersion, JsonContentType, "text/plain", ct);
}