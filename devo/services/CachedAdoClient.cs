using devo.models;
using devo.models.pullrequests;
using devo.models.threads;
using devo.models.workitems;

namespace devo.services;

/// <summary>Wraps an IAdoClient with in-memory TTL-based caching. Reads are
/// cached per-key; mutations delegate to the inner client and invalidate
/// affected prefixes only on success.</summary>
public sealed class CachedAdoClient : IAdoClient, ICacheInvalidator
{
    private static readonly TimeSpan? TtlSession = null; // never expires during session
    private static readonly TimeSpan? TtlLong = TimeSpan.FromMinutes(5);
    private static readonly TimeSpan? TtlMedium = TimeSpan.FromMinutes(1);
    private static readonly TimeSpan? TtlShort = TimeSpan.FromSeconds(30);
    private static readonly TimeSpan? TtlBrief = TimeSpan.FromSeconds(15);

    private readonly IAdoClient _inner;
    private readonly Func<DateTimeOffset> _now; // injectable for testing
    private readonly object _lock = new();
    private readonly Dictionary<string, CacheEntry> _store = [];

    public CachedAdoClient(IAdoClient inner)
        : this(inner, static () => DateTimeOffset.Now)
    {
    }

    internal CachedAdoClient(IAdoClient inner, Func<DateTimeOffset> now)
    {
        _inner = inner;
        _now = now;
    }

    private readonly record struct CacheEntry(object Value, DateTimeOffset? ExpiresAt);

    // --- Cache primitives ---

    private bool TryGet(string key, out object? value)
    {
        lock (_lock)
        {
            if (_store.TryGetValue(key, out CacheEntry entry)
                && (entry.ExpiresAt is null || _now() <= entry.ExpiresAt))
            {
                value = entry.Value;
                return true;
            }
        }
        value = null;
        return false;
    }

    private void Set(string key, object value, TimeSpan? ttl)
    {
        lock (_lock)
        {
            _store[key] = new CacheEntry(value, ttl is null ? null : _now() + ttl);
        }
    }

    public void InvalidateAll()
    {
        lock (_lock)
        {
            _store.Clear();
        }
    }

    public void InvalidatePrefix(string prefix)
    {
        lock (_lock)
        {
            foreach (string key in _store.Keys.Where(k => k.StartsWith(prefix, StringComparison.Ordinal)).ToList())
            {
                _store.Remove(key);
            }
        }
    }

    internal bool HasEntry(string key)
    {
        lock (_lock)
        {
            return _store.ContainsKey(key);
        }
    }

    private async Task<T> CachedAsync<T>(string key, TimeSpan? ttl, Func<Task<T>> fetch)
        where T : notnull
    {
        if (TryGet(key, out object? hit))
        {
            return (T)hit!;
        }
        T value = await fetch();
        Set(key, value, ttl);
        return value;
    }

    // --- Repositories & branches ---

    public Task<string> GetProjectIDAsync(CancellationToken ct = default) =>
        CachedAsync("projectid", TtlSession, () => _inner.GetProjectIDAsync(ct));

    public Task<IReadOnlyList<GitRepository>> ListRepositoriesAsync(CancellationToken ct = default) =>
        CachedAsync("repos", TtlLong, () => _inner.ListRepositoriesAsync(ct));

    public Task<IReadOnlyList<GitBranch>> ListBranchesAsync(string repoName, CancellationToken ct = default) =>
        CachedAsync($"branches:{repoName}", TtlMedium, () => _inner.ListBranchesAsync(repoName, ct));

    // --- Pull requests ---

    public Task<IReadOnlyList<PullRequest>> ListPullRequestsAsync(string status, CancellationToken ct = default) =>
        CachedAsync($"prs:{status}", TtlShort, () => _inner.ListPullRequestsAsync(status, ct));

    public Task<IReadOnlyList<PullRequest>> ListPullRequestsForRepoAsync(string repoName, string status, CancellationToken ct = default) =>
        CachedAsync($"prs:{repoName}:{status}", TtlShort, () => _inner.ListPullRequestsForRepoAsync(repoName, status, ct));

    public Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsAsync(CancellationToken ct = default) =>
        CachedAsync("prs:draft", TtlShort, () => _inner.ListDraftPullRequestsAsync(ct));

    public Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsForRepoAsync(string repoName, CancellationToken ct = default) =>
        CachedAsync($"prs:draft:{repoName}", TtlShort, () => _inner.ListDraftPullRequestsForRepoAsync(repoName, ct));

    public Task<PullRequest> GetPullRequestByIDAsync(int prID, CancellationToken ct = default) =>
        CachedAsync($"pr:id:{prID}", TtlBrief, () => _inner.GetPullRequestByIDAsync(prID, ct));

    public Task<PullRequest> GetPullRequestAsync(string repoID, int prID, CancellationToken ct = default) =>
        CachedAsync($"pr:{repoID}:{prID}", TtlBrief, () => _inner.GetPullRequestAsync(repoID, prID, ct));

    public Task<IReadOnlyList<CommentThread>> GetPullRequestThreadsAsync(string repoID, int prID, CancellationToken ct = default) =>
        CachedAsync($"threads:{repoID}:{prID}", TtlBrief, () => _inner.GetPullRequestThreadsAsync(repoID, prID, ct));

    public Task<string> GetCurrentUserIDAsync(CancellationToken ct = default) =>
        CachedAsync("userid", TtlSession, () => _inner.GetCurrentUserIDAsync(ct));

    // --- Pull request mutations: delegate then invalidate ---

    public async Task<PullRequest> CreatePullRequestAsync(string repoID, string title, string sourceBranch, string targetBranch, string? description, bool isDraft, CancellationToken ct = default)
    {
        PullRequest pr = await _inner.CreatePullRequestAsync(repoID, title, sourceBranch, targetBranch, description, isDraft, ct);
        InvalidatePrefix("prs:");
        return pr;
    }

    public async Task MergePullRequestAsync(string repoID, int prID, string strategy, string commitMsg, bool deleteSourceBranch, CancellationToken ct = default)
    {
        await _inner.MergePullRequestAsync(repoID, prID, strategy, commitMsg, deleteSourceBranch, ct);
        InvalidatePrefix("prs:");
        InvalidatePrefix("pr:");
    }

    public async Task AbandonPullRequestAsync(string repoID, int prID, CancellationToken ct = default)
    {
        await _inner.AbandonPullRequestAsync(repoID, prID, ct);
        InvalidatePrefix("prs:");
        InvalidatePrefix("pr:");
    }

    public async Task ToggleDraftAsync(string repoID, int prID, bool isDraft, CancellationToken ct = default)
    {
        await _inner.ToggleDraftAsync(repoID, prID, isDraft, ct);
        InvalidatePrefix("prs:");
        InvalidatePrefix("pr:");
    }

    public async Task CreateCommentThreadAsync(string repoID, int prID, string content, ThreadContext? threadContext = null, CancellationToken ct = default)
    {
        await _inner.CreateCommentThreadAsync(repoID, prID, content, threadContext, ct);
        InvalidatePrefix($"threads:{repoID}:{prID}");
    }

    public async Task ReplyToCommentThreadAsync(string repoID, int prID, int threadID, string content, CancellationToken ct = default)
    {
        await _inner.ReplyToCommentThreadAsync(repoID, prID, threadID, content, ct);
        InvalidatePrefix($"threads:{repoID}:{prID}");
    }

    public async Task UpdateCommentThreadStatusAsync(string repoID, int prID, int threadID, string status, CancellationToken ct = default)
    {
        await _inner.UpdateCommentThreadStatusAsync(repoID, prID, threadID, status, ct);
        InvalidatePrefix($"threads:{repoID}:{prID}");
    }

    public async Task SetVoteAsync(string repoID, int prID, string reviewerID, int vote, CancellationToken ct = default)
    {
        await _inner.SetVoteAsync(repoID, prID, reviewerID, vote, ct);
        InvalidatePrefix($"pr:{repoID}:{prID}");
        InvalidatePrefix("prs:");
    }

    // --- Iterations & diffs ---

    public Task<IReadOnlyList<Iteration>> GetPullRequestIterationsAsync(string repoID, int prID, CancellationToken ct = default) =>
        CachedAsync($"iterations:{repoID}:{prID}", TtlShort, () => _inner.GetPullRequestIterationsAsync(repoID, prID, ct));

    public Task<IReadOnlyList<IterationChange>> GetPullRequestIterationChangesAsync(string repoID, int prID, int iterationID, CancellationToken ct = default) =>
        // immutable per iteration
        CachedAsync($"iterchanges:{repoID}:{prID}:{iterationID}", TtlSession,
            () => _inner.GetPullRequestIterationChangesAsync(repoID, prID, iterationID, ct));

    public Task<string> GetFileContentAtCommitAsync(string repoID, string filePath, string commitID, CancellationToken ct = default) =>
        // immutable per commit
        CachedAsync($"file:{repoID}:{filePath}:{commitID}", TtlSession,
            () => _inner.GetFileContentAtCommitAsync(repoID, filePath, commitID, ct));

    public Task<string> BuildUnifiedDiffAsync(string repoID, IterationChange change, string oldCommitID, string newCommitID, CancellationToken ct = default) =>
        // immutable per commit pair
        CachedAsync($"diff:{repoID}:{change.Item.Path}:{oldCommitID}:{newCommitID}", TtlSession,
            () => _inner.BuildUnifiedDiffAsync(repoID, change, oldCommitID, newCommitID, ct));

    // --- Work items ---

    public Task<IReadOnlyList<WorkItem>> ListWorkItemsAsync(IReadOnlyList<string> types, string? assignedTo, string? areaPath, bool activeOnly, CancellationToken ct = default) =>
        CachedAsync($"wi:{string.Join(",", types)}:{assignedTo}:{areaPath}:{activeOnly}", TtlShort,
            () => _inner.ListWorkItemsAsync(types, assignedTo, areaPath, activeOnly, ct));

    public Task<WorkItem> GetWorkItemAsync(int id, CancellationToken ct = default) =>
        CachedAsync($"wi:{id}", TtlBrief, () => _inner.GetWorkItemAsync(id, ct));

    public Task<IReadOnlyList<WorkItemComment>> GetWorkItemCommentsAsync(int id, CancellationToken ct = default) =>
        CachedAsync($"wicomments:{id}", TtlBrief, () => _inner.GetWorkItemCommentsAsync(id, ct));

    public Task<IReadOnlyList<string>> GetWorkItemTypeStatesAsync(string workItemType, CancellationToken ct = default) =>
        CachedAsync($"wistates:{workItemType}", TtlSession, () => _inner.GetWorkItemTypeStatesAsync(workItemType, ct));

    // --- Work item mutations ---

    public async Task UpdateWorkItemStateAsync(int id, string state, CancellationToken ct = default)
    {
        await _inner.UpdateWorkItemStateAsync(id, state, ct);
        InvalidatePrefix("wi:");
    }

    public async Task AddWorkItemCommentAsync(int id, string text, CancellationToken ct = default)
    {
        await _inner.AddWorkItemCommentAsync(id, text, ct);
        InvalidatePrefix($"wicomments:{id}");
        InvalidatePrefix($"wi:{id}");
    }

    public async Task LinkWorkItemToPRAsync(int workItemID, string prArtifactURL, CancellationToken ct = default)
    {
        await _inner.LinkWorkItemToPRAsync(workItemID, prArtifactURL, ct);
        InvalidatePrefix($"wi:{workItemID}");
    }
}