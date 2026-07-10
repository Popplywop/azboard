using devo.models.pullrequests;
using devo.services;

namespace devo.tests;

// port of internal/api/cache_test.go
public class CachedAdoClientTests
{
    private static (CachedAdoClient Cache, CountingFakeClient Inner) Make()
    {
        var inner = new CountingFakeClient();
        return (new CachedAdoClient(inner), inner);
    }

    [Fact]
    public async Task CacheHit()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();

        var repos1 = await cache.ListRepositoriesAsync();
        var repos2 = await cache.ListRepositoriesAsync();

        Assert.Equal(1, inner.ListReposCalls);
        Assert.Equal(repos1.Count, repos2.Count);
    }

    [Fact]
    public async Task TtlExpiry()
    {
        var inner = new CountingFakeClient();
        DateTimeOffset now = DateTimeOffset.Now;
        var cache = new CachedAdoClient(inner, () => now);

        await cache.ListPullRequestsAsync("active"); // miss
        Assert.Equal(1, inner.ListPrsCalls);

        now += TimeSpan.FromSeconds(10); // within 30s TTL
        await cache.ListPullRequestsAsync("active"); // hit
        Assert.Equal(1, inner.ListPrsCalls);

        now += TimeSpan.FromSeconds(60); // well past TTL
        await cache.ListPullRequestsAsync("active"); // miss again
        Assert.Equal(2, inner.ListPrsCalls);
    }

    [Fact]
    public async Task InvalidateAll()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();

        await cache.ListRepositoriesAsync();
        await cache.ListPullRequestsAsync("active");

        cache.InvalidateAll();

        await cache.ListRepositoriesAsync();
        await cache.ListPullRequestsAsync("active");

        Assert.Equal(2, inner.ListReposCalls);
        Assert.Equal(2, inner.ListPrsCalls);
    }

    [Fact]
    public async Task InvalidatePrefix_OnlyMatchingKeys()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();

        await cache.ListRepositoriesAsync();
        await cache.ListPullRequestsAsync("active");

        cache.InvalidatePrefix("prs:");

        await cache.ListRepositoriesAsync();
        await cache.ListPullRequestsAsync("active");

        Assert.Equal(1, inner.ListReposCalls); // untouched
        Assert.Equal(2, inner.ListPrsCalls);   // invalidated
    }

    [Fact]
    public async Task SessionCache_NeverExpires()
    {
        var inner = new CountingFakeClient();
        DateTimeOffset now = DateTimeOffset.Now;
        var cache = new CachedAdoClient(inner, () => now);

        await cache.GetProjectIDAsync();

        now += TimeSpan.FromHours(1);
        await cache.GetProjectIDAsync();

        Assert.True(cache.HasEntry("projectid"));
    }

    [Fact]
    public async Task ConcurrentAccess_DoesNotThrow()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();

        await Task.WhenAll(Enumerable.Range(0, 50).Select(async _ =>
        {
            await cache.ListRepositoriesAsync();
            await cache.ListPullRequestsAsync("active");
        }));

        Assert.True(inner.ListReposCalls >= 1);
    }

    // --- per-method cache-hit round trips ---

    [Fact]
    public async Task CachedReads_ReturnSameData()
    {
        (CachedAdoClient cache, _) = Make();

        Assert.Equal(await cache.GetProjectIDAsync(), await cache.GetProjectIDAsync());
        Assert.Equal(await cache.GetCurrentUserIDAsync(), await cache.GetCurrentUserIDAsync());
        Assert.Equal((await cache.ListBranchesAsync("my-repo")).Count, (await cache.ListBranchesAsync("my-repo")).Count);
        Assert.Equal((await cache.GetPullRequestThreadsAsync("repo-1", 1847)).Count, (await cache.GetPullRequestThreadsAsync("repo-1", 1847)).Count);
        Assert.Equal((await cache.ListWorkItemsAsync(["Bug", "Task"], "@me", null, true)).Count, (await cache.ListWorkItemsAsync(["Bug", "Task"], "@me", null, true)).Count);
        Assert.Equal((await cache.GetWorkItemAsync(4521)).ID, (await cache.GetWorkItemAsync(4521)).ID);
        Assert.Equal((await cache.GetWorkItemCommentsAsync(4521)).Count, (await cache.GetWorkItemCommentsAsync(4521)).Count);
        Assert.Equal((await cache.GetWorkItemTypeStatesAsync("Bug")).Count, (await cache.GetWorkItemTypeStatesAsync("Bug")).Count);
        Assert.Equal((await cache.GetPullRequestIterationsAsync("repo-1", 1847)).Count, (await cache.GetPullRequestIterationsAsync("repo-1", 1847)).Count);
        Assert.Equal((await cache.GetPullRequestByIDAsync(1847)).PullRequestID, (await cache.GetPullRequestByIDAsync(1847)).PullRequestID);
        Assert.Equal((await cache.GetPullRequestAsync("repo-1", 1847)).PullRequestID, (await cache.GetPullRequestAsync("repo-1", 1847)).PullRequestID);
        Assert.Equal((await cache.ListPullRequestsForRepoAsync("inventory-api", "active")).Count, (await cache.ListPullRequestsForRepoAsync("inventory-api", "active")).Count);
        Assert.Equal((await cache.ListDraftPullRequestsAsync()).Count, (await cache.ListDraftPullRequestsAsync()).Count);
        Assert.Equal((await cache.ListDraftPullRequestsForRepoAsync("inventory-api")).Count, (await cache.ListDraftPullRequestsForRepoAsync("inventory-api")).Count);
        Assert.Equal(await cache.GetFileContentAtCommitAsync("repo-1", "/src/main.go", "abc123"), await cache.GetFileContentAtCommitAsync("repo-1", "/src/main.go", "abc123"));

        var change = new IterationChange
        {
            ChangeID = 1,
            ChangeType = "edit",
            Item = new ChangeItem { Path = "/src/main.go" },
        };
        Assert.Equal(
            await cache.BuildUnifiedDiffAsync("repo-1", change, "abc", "def"),
            await cache.BuildUnifiedDiffAsync("repo-1", change, "abc", "def"));
        Assert.Equal(
            (await cache.GetPullRequestIterationChangesAsync("repo-1", 1847, 1)).Count,
            (await cache.GetPullRequestIterationChangesAsync("repo-1", 1847, 1)).Count);
    }

    // --- mutation invalidation: success path (Go's MockClient couldn't test this) ---

    [Fact]
    public async Task SuccessfulMerge_InvalidatesPrCaches()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();

        await cache.ListPullRequestsAsync("active");
        Assert.Equal(1, inner.ListPrsCalls);

        await cache.MergePullRequestAsync("repo-1", 1847, "squash", "msg", deleteSourceBranch: true);

        await cache.ListPullRequestsAsync("active");
        Assert.Equal(2, inner.ListPrsCalls); // refetched after invalidation
    }

    [Fact]
    public async Task SuccessfulCreateThread_InvalidatesThatPrsThreads()
    {
        (CachedAdoClient cache, _) = Make();

        await cache.GetPullRequestThreadsAsync("repo-1", 1847);
        await cache.GetPullRequestThreadsAsync("repo-2", 99);
        Assert.True(cache.HasEntry("threads:repo-1:1847"));
        Assert.True(cache.HasEntry("threads:repo-2:99"));

        await cache.CreateCommentThreadAsync("repo-1", 1847, "test");

        Assert.False(cache.HasEntry("threads:repo-1:1847"));
        Assert.True(cache.HasEntry("threads:repo-2:99")); // other PR untouched
    }

    [Fact]
    public async Task SuccessfulUpdateWorkItemState_InvalidatesWorkItems()
    {
        (CachedAdoClient cache, _) = Make();

        await cache.GetWorkItemAsync(4521);
        Assert.True(cache.HasEntry("wi:4521"));

        await cache.UpdateWorkItemStateAsync(4521, "Active");

        Assert.False(cache.HasEntry("wi:4521"));
    }

    [Fact]
    public async Task SuccessfulSetVote_InvalidatesPrAndLists()
    {
        (CachedAdoClient cache, _) = Make();

        await cache.GetPullRequestAsync("repo-1", 1847);
        await cache.ListPullRequestsAsync("active");
        Assert.True(cache.HasEntry("pr:repo-1:1847"));
        Assert.True(cache.HasEntry("prs:active"));

        await cache.SetVoteAsync("repo-1", 1847, "user-1", 10);

        Assert.False(cache.HasEntry("pr:repo-1:1847"));
        Assert.False(cache.HasEntry("prs:active"));
    }

    // --- mutation invalidation: failure path keeps cache (Go: err != nil) ---

    [Fact]
    public async Task FailedMutation_DoesNotInvalidate()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();
        await cache.GetWorkItemAsync(4521);
        inner.FailMutations = true;

        await Assert.ThrowsAsync<InvalidOperationException>(
            () => cache.UpdateWorkItemStateAsync(4521, "Active"));

        Assert.True(cache.HasEntry("wi:4521"));
    }

    [Fact]
    public async Task FailedMerge_KeepsPrCache()
    {
        (CachedAdoClient cache, CountingFakeClient inner) = Make();
        await cache.ListPullRequestsAsync("active");
        inner.FailMutations = true;

        await Assert.ThrowsAsync<InvalidOperationException>(
            () => cache.MergePullRequestAsync("repo-1", 1, "squash", "msg", deleteSourceBranch: true));

        Assert.True(cache.HasEntry("prs:active"));
        Assert.Equal(1, inner.ListPrsCalls);
    }
}