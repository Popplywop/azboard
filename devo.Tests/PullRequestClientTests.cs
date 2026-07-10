using System.Text.Json;

using devo.exceptions;
using devo.models.pullrequests;
using devo.services;

namespace devo.tests;

// port of internal/api/pullrequests_test.go
public class PullRequestClientTests
{
    [Fact]
    public async Task ListPullRequests()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests",
            TestServer.Json(200, TestData.ListResponse(TestData.PullRequest(42, "Test PR"))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<PullRequest> prs = await client.ListPullRequestsAsync("active");

        Assert.Single(prs);
        Assert.Equal(42, prs[0].PullRequestID);
        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Get, req.Method);
        Assert.Equal("active", req.Query["searchCriteria.status"]);
    }

    [Fact]
    public async Task ListPullRequestsForRepo()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/my-repo/pullrequests",
            TestServer.Json(200, TestData.ListResponse(TestData.PullRequest(10, "Repo PR"))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<PullRequest> prs = await client.ListPullRequestsForRepoAsync("my-repo", "active");

        Assert.Single(prs);
        Assert.Equal(10, prs[0].PullRequestID);
    }

    [Fact]
    public async Task ListDraftPullRequests()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests",
            TestServer.Json(200, TestData.ListResponse(TestData.PullRequest(99, isDraft: true))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<PullRequest> prs = await client.ListDraftPullRequestsAsync();

        Assert.Single(prs);
        Assert.True(prs[0].IsDraft);
        Assert.Equal("true", handler.Requests[0].Query["searchCriteria.isDraft"]);
    }

    [Fact]
    public async Task ListDraftPullRequestsForRepo()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/my-repo/pullrequests",
            TestServer.Json(200, TestData.ListResponse(TestData.PullRequest(77, isDraft: true))));
        AdoClient client = TestServer.Client(handler);

        IReadOnlyList<PullRequest> prs = await client.ListDraftPullRequestsForRepoAsync("my-repo");

        Assert.Single(prs);
        Assert.Equal("true", handler.Requests[0].Query["searchCriteria.isDraft"]);
    }

    [Fact]
    public async Task GetPullRequestByID_FiltersClientSide()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Json(200, TestData.ListResponse(
            TestData.PullRequest(1, "Wrong PR"),
            TestData.PullRequest(42, "Right PR"))));
        AdoClient client = TestServer.Client(handler);

        PullRequest pr = await client.GetPullRequestByIDAsync(42);

        Assert.Equal("Right PR", pr.Title);
    }

    [Fact]
    public async Task GetPullRequestByID_NotFound()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Json(200, TestData.ListResponse()));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.GetPullRequestByIDAsync(999));
        Assert.Contains("not found", ex.Message);
    }

    [Fact]
    public async Task GetPullRequest()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5",
            TestServer.Json(200, TestData.PullRequest(5, "Single PR")));
        AdoClient client = TestServer.Client(handler);

        PullRequest pr = await client.GetPullRequestAsync("repo-1", 5);

        Assert.Equal(5, pr.PullRequestID);
    }

    [Fact]
    public async Task GetPullRequestThreads()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/threads",
            TestServer.Json(200, TestData.ListResponse(new { id = 1, status = "active" })));
        AdoClient client = TestServer.Client(handler);

        var threads = await client.GetPullRequestThreadsAsync("repo-1", 5);

        Assert.Single(threads);
        Assert.Equal(1, threads[0].ID);
    }

    [Fact]
    public async Task CreatePullRequest()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests",
            TestServer.Json(201, TestData.PullRequest(100, "New PR")));
        AdoClient client = TestServer.Client(handler);

        PullRequest pr = await client.CreatePullRequestAsync(
            "repo-1", "New PR", "refs/heads/feature", "refs/heads/main", "desc", isDraft: false);

        Assert.Equal(100, pr.PullRequestID);
        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Post, req.Method);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        Assert.Equal("New PR", body.RootElement.GetProperty("title").GetString());
        Assert.Equal("refs/heads/feature", body.RootElement.GetProperty("sourceRefName").GetString());
    }

    [Fact]
    public async Task MergePullRequest()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.MergePullRequestAsync("repo-1", 5, "squash", "merge commit", deleteSourceBranch: true);

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Patch, req.Method);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        Assert.Equal("completed", body.RootElement.GetProperty("status").GetString());
        Assert.Equal("squash", body.RootElement
            .GetProperty("completionOptions").GetProperty("mergeStrategy").GetString());
    }

    [Fact]
    public async Task AbandonPullRequest()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.AbandonPullRequestAsync("repo-1", 5);

        using JsonDocument body = JsonDocument.Parse(handler.Requests[0].Body);
        Assert.Equal("abandoned", body.RootElement.GetProperty("status").GetString());
    }

    [Fact]
    public async Task ToggleDraft()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.ToggleDraftAsync("repo-1", 5, isDraft: true);

        Assert.Contains("\"isDraft\":true", handler.Requests[0].Body);
    }

    [Fact]
    public async Task CreateCommentThread()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/threads", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.CreateCommentThreadAsync("repo-1", 5, "looks good");

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Post, req.Method);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        JsonElement comments = body.RootElement.GetProperty("comments");
        Assert.Equal(1, comments.GetArrayLength());
        Assert.Equal("looks good", comments[0].GetProperty("content").GetString());
        // no threadContext arg -> field omitted entirely (Go omitempty parity)
        Assert.False(body.RootElement.TryGetProperty("threadContext", out _));
    }

    [Fact]
    public async Task ReplyToCommentThread()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/threads/1/comments", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.ReplyToCommentThreadAsync("repo-1", 5, 1, "thanks");

        Assert.Equal(HttpMethod.Post, handler.Requests[0].Method);
    }

    [Fact]
    public async Task UpdateCommentThreadStatus()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/threads/1", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.UpdateCommentThreadStatusAsync("repo-1", 5, 1, "fixed");

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Patch, req.Method);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        Assert.Equal("fixed", body.RootElement.GetProperty("status").GetString());
    }

    [Fact]
    public async Task SetVote()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/repositories/repo-1/pullrequests/5/reviewers/user-1", TestServer.Status(200));
        AdoClient client = TestServer.Client(handler);

        await client.SetVoteAsync("repo-1", 5, "user-1", 10);

        CapturedRequest req = handler.Requests[0];
        Assert.Equal(HttpMethod.Put, req.Method);
        using JsonDocument body = JsonDocument.Parse(req.Body);
        Assert.Equal(10, body.RootElement.GetProperty("vote").GetInt32());
    }

    [Fact]
    public async Task GetCurrentUserID()
    {
        var handler = new FakeHandler();
        handler.Handle("/connectionData", TestServer.Json(200, new
        {
            authenticatedUser = new { id = "user-abc", providerDisplayName = "Test User" },
        }));
        AdoClient client = TestServer.Client(handler);

        string id = await client.GetCurrentUserIDAsync();

        Assert.Equal("user-abc", id);
    }

    [Fact]
    public async Task ListPullRequests_404()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Json(404, new { message = "not found" }));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.ListPullRequestsAsync("active"));
        Assert.True(ex.IsNotFound);
    }

    [Fact]
    public async Task ListPullRequests_500()
    {
        var handler = new FakeHandler();
        handler.Handle("/git/pullrequests", TestServer.Json(500, new { message = "internal error" }));
        AdoClient client = TestServer.Client(handler);

        ApiException ex = await Assert.ThrowsAsync<ApiException>(
            () => client.ListPullRequestsAsync("active"));
        Assert.True(ex.IsServerError);
    }
}