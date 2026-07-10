using devo.exceptions;
using devo.services;

namespace devo.tests;

// port of internal/api/client_test.go — request plumbing
public class AdoClientTests
{
    [Fact]
    public async Task Non2xx_ThrowsApiExceptionWithStatusAndBody()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Text(403, "forbidden"));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.ListPullRequestsAsync("active"));
        Assert.Equal(403, ex.StatusCode);
        Assert.Equal("forbidden", ex.Body);
    }

    [Fact]
    public async Task RateLimit429_SetsIsRateLimited()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Text(429, "too many requests"));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.ListPullRequestsAsync("active"));
        Assert.True(ex.IsRateLimited);
    }

    [Fact]
    public async Task GetContent_Success_SendsTextPlainAccept()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo/items", TestServer.Text(200, "file content here"));
        AdoClient client = TestServer.Client(handler);

        string content = await client.GetFileContentAtCommitAsync("repo", "/file.go", "abc123");

        Assert.Equal("file content here", content);
        Assert.Equal("text/plain", handler.Requests[0].Accept);
    }

    [Fact]
    public async Task GetContent_404_SetsIsNotFound()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo/items", TestServer.Text(404, "not found"));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.GetFileContentAtCommitAsync("repo", "/missing.go", "abc123"));
        Assert.True(ex.IsNotFound);
    }

    [Fact]
    public async Task PatAuth_401_DoesNotRetry()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Status(401));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.ListPullRequestsAsync("active"));
        Assert.Equal(401, ex.StatusCode);
        Assert.Single(handler.Requests);
    }

    [Fact]
    public async Task Requests_AppendApiVersion()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories", TestServer.Json(200, TestData.ListResponse()));
        AdoClient client = TestServer.Client(handler);

        await client.ListRepositoriesAsync();

        Assert.Equal("7.1", handler.Requests[0].Query["api-version"]);
    }

    [Fact]
    public async Task Requests_SendBasicAuthHeader()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories", TestServer.Json(200, TestData.ListResponse()));
        AdoClient client = TestServer.Client(handler);

        await client.ListRepositoriesAsync();

        string expected = "Basic " + Convert.ToBase64String(
            System.Text.Encoding.UTF8.GetBytes(":test-pat-token"));
        Assert.Equal(expected, handler.Requests[0].Authorization);
    }
}