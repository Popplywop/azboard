using System.Text.Json;
using System.Text.Json.Serialization;

using devo.exceptions;

using devo.models.config;

namespace devo.config;

public sealed record Config
{
    public required AuthMethod AuthMethod { get; init; }

    public string? OrgUrl { get; init; }
    public required string Org { get; init; }
    public required string Project { get; init; }
    public string? Pat { get; init; }

    public IReadOnlyList<string>? Repos { get; init; }

    public IReadOnlyList<string> WorkItemTypes { get; init; } =
        ["User Story", "Bug", "Task", "Feature", "Epic"];

    public string DefaultMergeStrategy { get; init; } = "squash";

    public string? AreaPath { get; init; }
    public int AutoRefreshSeconds { get; init; }

    public static async Task<Config> LoadAsync(CancellationToken ct = default)
        => await LoadFromFileAsync(ConfigFilePath(), ct);

    /// <summary>
    /// Reads the existing config, updates only the repos field, and writes back.
    /// </summary>
    public static async Task UpdateReposAsync(IReadOnlyList<string> repos, CancellationToken ct = default)
    {
        var path = ConfigFilePath();

        var raw = await ReadConfigJsonAsync(path, ct).ConfigureAwait(false);
        raw.Repos = repos.Count > 0 ? [.. repos] : null;

        string json;
        try
        {
            json = JsonSerializer.Serialize(raw, WriteOptions);
        }
        catch (Exception e)
        {
            throw new ConfigException($"failed to marshal config: {e.Message}", e);
        }

        await File.WriteAllTextAsync(path, json + "\n", ct).ConfigureAwait(false);
        if (!OperatingSystem.IsWindows())
        {
            File.SetUnixFileMode(path, UnixFileMode.UserRead | UnixFileMode.UserWrite);
        }

    }

    internal static async Task<Config> LoadFromFileAsync(string path, CancellationToken ct = default)
    {
        var raw = await ReadConfigJsonAsync(path, ct).ConfigureAwait(false);
        var authMethod = raw.AuthMethod?.ToLowerInvariant() switch
        {
            "pat" => AuthMethod.Pat,
            "azcli" or null or "" => AuthMethod.AzCli,
            var m => throw new ConfigException(
                $"unknown auth method \"{m}\": must be \"pat\" or \"azcli\""),
        };

        var org = "";
        var project = "";

        if (!string.IsNullOrEmpty(raw.OrgUrl))
        {
            var (parsedOrg, parsedProject) = ParseOrgUrl(raw.OrgUrl);
            if (parsedOrg != string.Empty)
            {
                org = parsedOrg;
            }

            if (parsedProject != string.Empty)
            {
                project = parsedProject;
            }
        }

        if (!string.IsNullOrEmpty(raw.Project))
        {
            project = raw.Project;
        }

        // PAT
        string? pat = null;
        if (authMethod == AuthMethod.Pat)
        {
            pat = raw.Pat;
            if (string.IsNullOrEmpty(pat))
            {
                throw new ConfigException($"PAT auth requires \"pat\" field in {path}");
            }
        }

        // AutoRefreshSeconds — 0 stays disabled, otherwise enforce minimum of 10
        var autoRefresh = raw.AutoRefreshSeconds switch
        {
            <= 0 => 0,
            < 10 => 10,
            var s => s,
        };

        // Validate
        if (org == string.Empty)
        {
            throw new ConfigException($"organization is required: set \"org_url\" in {path}");
        }

        if (project == string.Empty)
        {
            throw new ConfigException($"project is required: set \"project\" in {path}");
        }

        var cfg = new Config
        {
            AuthMethod = authMethod,
            OrgUrl = raw.OrgUrl,
            Org = org,
            Project = project,
            Pat = pat,
            Repos = raw.Repos ?? [],
            AreaPath = raw.AreaPath,
            AutoRefreshSeconds = autoRefresh,
        };

        if (raw.WorkItemTypes is { Count: > 0 })
        {
            cfg = cfg with { WorkItemTypes = raw.WorkItemTypes };
        }

        if (raw.DefaultMergeStrategy is "squash" or "merge" or "rebase" or "semilinear")
        {
            cfg = cfg with { DefaultMergeStrategy = raw.DefaultMergeStrategy };
        }

        return cfg;
    }

    internal static (string Org, string Project) ParseOrgUrl(string rawUrl)
    {
        if (!Uri.TryCreate(rawUrl.TrimEnd('/'), UriKind.Absolute, out var uri))
        {

            return ("", "");
        }


        var parts = uri.AbsolutePath.Trim('/').Split('/');

        if (uri.Host.EndsWith(".visualstudio.com", StringComparison.OrdinalIgnoreCase))
        {
            var org = uri.Host[..^".visualstudio.com".Length];
            var project = parts.Length > 0 && parts[0] != "" ? parts[0] : "";
            return (org, project);
        }

        if (string.Equals(uri.Host, "dev.azure.com", StringComparison.OrdinalIgnoreCase))
        {
            var org = parts.Length > 0 ? parts[0] : "";
            var project = parts.Length > 1 ? parts[1] : "";
            return (org, project);
        }

        return ("", "");
    }

    internal static string ConfigFilePath() =>
        Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.UserProfile),
            ".config", "devo", "config.json");

    private static async Task<ConfigJson> ReadConfigJsonAsync(string path, CancellationToken ct)
    {
        FileStream data;
        try
        {
            data = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.Read,
                bufferSize: 4096, useAsync: true);
        }
        catch (Exception e) when (e is IOException or UnauthorizedAccessException)
        {
            throw new ConfigException($"could not read config file: {e.Message}", e);
        }

        await using (data.ConfigureAwait(false))
        {
            try
            {
                return await JsonSerializer.DeserializeAsync<ConfigJson>(data, cancellationToken: ct).ConfigureAwait(false)
                    ?? new ConfigJson();
            }
            catch (JsonException e)
            {
                throw new ConfigException($"invalid JSON in config file: {e.Message}", e);
            }
        }
    }

    private static readonly JsonSerializerOptions WriteOptions = new()
    {
        WriteIndented = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingDefault,
    };
}