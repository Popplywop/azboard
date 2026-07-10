using devo.config;
using devo.exceptions;
using devo.models.config;

namespace devo.tests;

// port of internal/config/config_test.go
public class ConfigTests : IDisposable
{
    private readonly string _dir;

    public ConfigTests()
    {
        _dir = Path.Combine(Path.GetTempPath(), "devo-tests-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_dir);
    }

    public void Dispose() => Directory.Delete(_dir, recursive: true);

    private async Task<string> WriteConfigAsync(string content)
    {
        string path = Path.Combine(_dir, "config.json");
        await File.WriteAllTextAsync(path, content);
        return path;
    }

    [Fact]
    public async Task Load_ValidAzCli()
    {
        string path = await WriteConfigAsync("""
            {
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject"
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal(AuthMethod.AzCli, cfg.AuthMethod);
        Assert.Equal("myorg", cfg.Org);
        Assert.Equal("MyProject", cfg.Project);
    }

    [Fact]
    public async Task Load_ValidPat()
    {
        string path = await WriteConfigAsync("""
            {
                "auth_method": "pat",
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject",
                "pat": "secret-token"
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal(AuthMethod.Pat, cfg.AuthMethod);
        Assert.Equal("secret-token", cfg.Pat);
    }

    [Fact]
    public async Task Load_PatMissingToken_Throws()
    {
        string path = await WriteConfigAsync("""
            {
                "auth_method": "pat",
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject"
            }
            """);
        await Assert.ThrowsAsync<ConfigException>(() => Config.LoadFromFileAsync(path));
    }

    [Fact]
    public async Task Load_MissingOrg_Throws()
    {
        string path = await WriteConfigAsync("""{ "project": "MyProject" }""");
        await Assert.ThrowsAsync<ConfigException>(() => Config.LoadFromFileAsync(path));
    }

    [Fact]
    public async Task Load_MissingProject_Throws()
    {
        string path = await WriteConfigAsync("""{ "org_url": "https://dev.azure.com/myorg" }""");
        await Assert.ThrowsAsync<ConfigException>(() => Config.LoadFromFileAsync(path));
    }

    [Fact]
    public async Task Load_InvalidAuthMethod_Throws()
    {
        string path = await WriteConfigAsync("""
            {
                "auth_method": "oauth",
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject"
            }
            """);
        await Assert.ThrowsAsync<ConfigException>(() => Config.LoadFromFileAsync(path));
    }

    [Fact]
    public async Task Load_DefaultValues()
    {
        string path = await WriteConfigAsync("""
            {
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject"
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal("squash", cfg.DefaultMergeStrategy);
        Assert.Equal(5, cfg.WorkItemTypes.Count);
    }

    [Fact]
    public async Task Load_CustomValues()
    {
        string path = await WriteConfigAsync("""
            {
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject",
                "repos": ["repo1", "repo2"],
                "work_item_types": ["Bug", "Task"],
                "default_merge_strategy": "rebase",
                "area_path": "Project\\Team"
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal(["repo1", "repo2"], cfg.Repos);
        Assert.Equal(2, cfg.WorkItemTypes.Count);
        Assert.Equal("rebase", cfg.DefaultMergeStrategy);
        Assert.Equal("Project\\Team", cfg.AreaPath);
    }

    [Fact]
    public async Task Load_ProjectFromUrl()
    {
        string path = await WriteConfigAsync("""{ "org_url": "https://dev.azure.com/myorg/InlineProject" }""");
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal("InlineProject", cfg.Project);
    }

    [Fact]
    public async Task Load_ExplicitProjectOverridesUrl()
    {
        string path = await WriteConfigAsync("""
            {
                "org_url": "https://dev.azure.com/myorg/FromURL",
                "project": "Explicit"
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal("Explicit", cfg.Project);
    }

    [Theory]
    [InlineData("https://dev.azure.com/myorg", "myorg", "")]
    [InlineData("https://dev.azure.com/myorg/myproject", "myorg", "myproject")]
    [InlineData("https://myorg.visualstudio.com", "myorg", "")]
    [InlineData("https://myorg.visualstudio.com/myproject", "myorg", "myproject")]
    [InlineData("https://dev.azure.com/myorg/", "myorg", "")]
    [InlineData("https://example.com/foo", "", "")]
    public void ParseOrgUrl(string url, string wantOrg, string wantProject)
    {
        (string org, string project) = Config.ParseOrgUrl(url);
        Assert.Equal(wantOrg, org);
        Assert.Equal(wantProject, project);
    }

    [Fact]
    public void ConfigFilePath_EndsWithConfigJson()
    {
        string path = Config.ConfigFilePath();
        Assert.True(Path.IsPathRooted(path));
        Assert.Equal("config.json", Path.GetFileName(path));
    }

    [Fact]
    public async Task Load_InvalidJson_Throws()
    {
        string path = await WriteConfigAsync("not json");
        await Assert.ThrowsAsync<ConfigException>(() => Config.LoadFromFileAsync(path));
    }

    [Fact]
    public async Task Load_NonexistentFile_Throws() =>
        await Assert.ThrowsAsync<ConfigException>(
            () => Config.LoadFromFileAsync("/nonexistent/path/config.json"));

    [Theory]
    [InlineData("", 0)]                              // absent -> disabled
    [InlineData(""" ,"auto_refresh_seconds": 30""", 30)]
    [InlineData(""" ,"auto_refresh_seconds": 3""", 10)] // clamped to minimum
    public async Task Load_AutoRefreshSeconds(string extraJson, int want)
    {
        string path = await WriteConfigAsync($$"""
            {
                "org_url": "https://dev.azure.com/myorg",
                "project": "MyProject"{{extraJson}}
            }
            """);
        Config cfg = await Config.LoadFromFileAsync(path);

        Assert.Equal(want, cfg.AutoRefreshSeconds);
    }
}