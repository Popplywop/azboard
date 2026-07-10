using System.Net;
using System.Text;
using System.Text.Json;

using devo.config;
using devo.models.config;
using devo.services;

namespace devo.tests;

/// <summary>A captured HTTP request. Path is relative to the _apis root
/// (org- and project-level prefixes stripped), mirroring the Go httptest
/// handlers.</summary>
public sealed record CapturedRequest(
    HttpMethod Method,
    string Path,
    IReadOnlyDictionary<string, string> Query,
    string Body,
    string? ContentType,
    string Accept,
    string? Authorization);

/// <summary>In-memory stand-in for Go's httptest.Server: routes by path,
/// records every request for post-call assertions.</summary>
public sealed class FakeHandler : HttpMessageHandler
{
    private const string ProjectPrefix = "/TestProject/_apis";
    private const string OrgPrefix = "/_apis";

    private readonly Dictionary<string, Func<CapturedRequest, HttpResponseMessage>> _routes = [];

    public List<CapturedRequest> Requests { get; } = [];

    /// <summary>Registers a handler for a path. A path ending in '/' matches
    /// as a prefix, like Go's ServeMux.</summary>
    public void Handle(string path, Func<CapturedRequest, HttpResponseMessage> handler) =>
        _routes[path] = handler;

    /// <summary>Registers a canned response for a path.</summary>
    public void Handle(string path, HttpResponseMessage response) =>
        _routes[path] = _ => response;

    protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        string raw = request.RequestUri!.AbsolutePath;
        string path = raw.StartsWith(ProjectPrefix, StringComparison.Ordinal)
            ? raw[ProjectPrefix.Length..]
            : raw.StartsWith(OrgPrefix, StringComparison.Ordinal) ? raw[OrgPrefix.Length..] : raw;

        string body = request.Content is null
            ? ""
            : await request.Content.ReadAsStringAsync(cancellationToken);

        var captured = new CapturedRequest(
            request.Method,
            path,
            ParseQuery(request.RequestUri.Query),
            body,
            request.Content?.Headers.ContentType?.MediaType,
            request.Headers.Accept.ToString(),
            request.Headers.TryGetValues("Authorization", out IEnumerable<string>? auth)
                ? string.Join(",", auth)
                : null);
        Requests.Add(captured);

        if (_routes.TryGetValue(path, out Func<CapturedRequest, HttpResponseMessage>? handler))
        {
            return handler(captured);
        }
        foreach ((string route, Func<CapturedRequest, HttpResponseMessage> h) in _routes)
        {
            if (route.EndsWith('/') && path.StartsWith(route, StringComparison.Ordinal))
            {
                return h(captured);
            }
        }
        return new HttpResponseMessage(HttpStatusCode.NotFound)
        {
            Content = new StringContent($"no test route for {path}"),
        };
    }

    private static Dictionary<string, string> ParseQuery(string query)
    {
        var result = new Dictionary<string, string>();
        foreach (string pair in query.TrimStart('?').Split('&', StringSplitOptions.RemoveEmptyEntries))
        {
            string[] kv = pair.Split('=', 2);
            result[Uri.UnescapeDataString(kv[0])] = kv.Length > 1 ? Uri.UnescapeDataString(kv[1]) : "";
        }
        return result;
    }
}

public static class TestServer
{
    /// <summary>Creates an AdoClient (PAT auth, project "TestProject") whose
    /// HTTP goes to the given fake handler.</summary>
    public static AdoClient Client(FakeHandler handler)
    {
        var config = new Config
        {
            AuthMethod = AuthMethod.Pat,
            Org = "testorg",
            OrgUrl = "https://unit.test",
            Project = "TestProject",
            Pat = "test-pat-token",
        };
        return new AdoClient(config, new HttpClient(handler));
    }

    public static HttpResponseMessage Json(int status, object body) =>
        new((HttpStatusCode)status)
        {
            Content = new StringContent(
                JsonSerializer.Serialize(body), Encoding.UTF8, "application/json"),
        };

    public static HttpResponseMessage Text(int status, string body) =>
        new((HttpStatusCode)status)
        {
            Content = new StringContent(body, Encoding.UTF8, "text/plain"),
        };

    public static HttpResponseMessage Status(int status) => new((HttpStatusCode)status);
}

/// <summary>Builders for response JSON shapes that satisfy the models'
/// required properties.</summary>
public static class TestData
{
    public static object Identity(string id = "user-1") =>
        new { displayName = "Test User", uniqueName = "user@example.com", id };

    public static object PullRequest(int id, string title = "PR", string status = "active", bool isDraft = false) =>
        new
        {
            pullRequestId = id,
            title,
            description = "",
            status,
            creationDate = "2024-01-01T00:00:00Z",
            sourceRefName = "refs/heads/feature",
            targetRefName = "refs/heads/main",
            mergeStatus = "succeeded",
            isDraft,
            createdBy = Identity(),
            repository = new { id = "repo-1", name = "test-repo" },
            reviewers = Array.Empty<object>(),
        };

    public static object ListResponse(params object[] items) =>
        new { count = items.Length, value = items };
}