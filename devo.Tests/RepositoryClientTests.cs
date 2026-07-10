using devo.models;
using devo.models.pullrequests;
using devo.services;

namespace devo.tests;

// port of internal/api/repos_test.go
public class RepositoryClientTests
{
    [Fact]
    public async Task ListRepositories()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories", TestServer.Json(200, TestData.ListResponse(
            new { id = "r1", name = "inventory-api" },
            new { id = "r2", name = "web-portal" })));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<GitRepository> repos = await client.ListRepositoriesAsync();

        Assert.Equal(HttpMethod.Get, handler.Requests[0].Method);
        Assert.Equal(2, repos.Count);
        Assert.Equal("inventory-api", repos[0].Name);
    }

    [Fact]
    public async Task ListBranches()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/my-repo/refs", TestServer.Json(200, TestData.ListResponse(
            new { name = "refs/heads/main" },
            new { name = "refs/heads/feature" })));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<GitBranch> branches = await client.ListBranchesAsync("my-repo");

        Assert.Equal("heads", handler.Requests[0].Query["filter"]);
        Assert.Equal(2, branches.Count);
        Assert.Equal("main", branches[0].ShortName);
    }

    [Fact]
    public async Task GetProjectID()
    {
        var handler = new FakeHandler();
        handler.Handle("/projects/TestProject",
            TestServer.Json(200, new { id = "proj-123", name = "TestProject" }));
        AdoClient client = TestServer.Client(handler);

        string id = await client.GetProjectIDAsync();

        Assert.Equal("proj-123", id);
    }
}