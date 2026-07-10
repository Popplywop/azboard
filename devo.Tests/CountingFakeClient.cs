using devo.models;
using devo.models.pullrequests;
using devo.models.threads;
using devo.models.workitems;
using devo.services;

namespace devo.tests;

/// <summary>IAdoClient test double: canned data, per-method call counters,
/// and optional mutation failure (C# stand-in for Go's countingClient over
/// MockClient).</summary>
public sealed class CountingFakeClient : IAdoClient
{
    public int ListReposCalls;
    public int ListPrsCalls;
    public int Mutations;

    /// <summary>When true, every mutation throws instead of succeeding.</summary>
    public bool FailMutations { get; set; }

    private static readonly IdentityRef User = new() { DisplayName = "Test", UniqueName = "t@e.com", ID = "user-1" };

    private static PullRequest Pr(int id) => new()
    {
        PullRequestID = id,
        Title = $"PR {id}",
        Status = "active",
        SourceRefName = "refs/heads/feature",
        TargetRefName = "refs/heads/main",
        CreatedBy = User,
        Repository = new GitRepository { ID = "repo-1", Name = "test-repo" },
    };

    private static WorkItem Wi(int id) => new()
    {
        ID = id,
        URL = $"http://example.com/{id}",
        Fields = new WorkItemFields { Title = $"Item {id}", State = "Active", WorkItemType = "Bug" },
    };

    private void Mutate()
    {
        Mutations++;
        if (FailMutations)
        {
            throw new InvalidOperationException("mutation failed (fake)");
        }
    }

    public Task<string> GetProjectIDAsync(CancellationToken ct = default) =>
        Task.FromResult("proj-123");

    public Task<IReadOnlyList<GitRepository>> ListRepositoriesAsync(CancellationToken ct = default)
    {
        ListReposCalls++;
        return Task.FromResult<IReadOnlyList<GitRepository>>(
            [new GitRepository { ID = "r1", Name = "inventory-api" }]);
    }

    public Task<IReadOnlyList<GitBranch>> ListBranchesAsync(string repoName, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<GitBranch>>([new GitBranch { Name = "refs/heads/main" }]);

    public Task<IReadOnlyList<PullRequest>> ListPullRequestsAsync(string status, CancellationToken ct = default)
    {
        ListPrsCalls++;
        return Task.FromResult<IReadOnlyList<PullRequest>>([Pr(1847)]);
    }

    public Task<IReadOnlyList<PullRequest>> ListPullRequestsForRepoAsync(string repoName, string status, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<PullRequest>>([Pr(1)]);

    public Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsAsync(CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<PullRequest>>([Pr(2)]);

    public Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsForRepoAsync(string repoName, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<PullRequest>>([Pr(3)]);

    public Task<PullRequest> GetPullRequestByIDAsync(int prID, CancellationToken ct = default) =>
        Task.FromResult(Pr(prID));

    public Task<PullRequest> GetPullRequestAsync(string repoID, int prID, CancellationToken ct = default) =>
        Task.FromResult(Pr(prID));

    public Task<IReadOnlyList<CommentThread>> GetPullRequestThreadsAsync(string repoID, int prID, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<CommentThread>>([new CommentThread { ID = 1, Status = "active" }]);

    public Task<string> GetCurrentUserIDAsync(CancellationToken ct = default) =>
        Task.FromResult("user-1");

    public Task<PullRequest> CreatePullRequestAsync(string repoID, string title, string sourceBranch, string targetBranch, string? description, bool isDraft, CancellationToken ct = default)
    {
        Mutate();
        return Task.FromResult(Pr(100));
    }

    public Task MergePullRequestAsync(string repoID, int prID, string strategy, string commitMsg, bool deleteSourceBranch, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task AbandonPullRequestAsync(string repoID, int prID, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task ToggleDraftAsync(string repoID, int prID, bool isDraft, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task CreateCommentThreadAsync(string repoID, int prID, string content, ThreadContext? threadContext = null, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task ReplyToCommentThreadAsync(string repoID, int prID, int threadID, string content, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task UpdateCommentThreadStatusAsync(string repoID, int prID, int threadID, string status, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task SetVoteAsync(string repoID, int prID, string reviewerID, int vote, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task<IReadOnlyList<Iteration>> GetPullRequestIterationsAsync(string repoID, int prID, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<Iteration>>([new Iteration
        {
            ID = 1,
            SourceRefCommit = new CommitRef { CommitID = "abc" },
            TargetRefCommit = new CommitRef { CommitID = "def" },
        }]);

    public Task<IReadOnlyList<IterationChange>> GetPullRequestIterationChangesAsync(string repoID, int prID, int iterationID, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<IterationChange>>([new IterationChange
        {
            ChangeID = 1,
            ChangeType = "edit",
            Item = new ChangeItem { Path = "/src/main.go" },
        }]);

    public Task<string> GetFileContentAtCommitAsync(string repoID, string filePath, string commitID, CancellationToken ct = default) =>
        Task.FromResult("file content");

    public Task<string> BuildUnifiedDiffAsync(string repoID, IterationChange change, string oldCommitID, string newCommitID, CancellationToken ct = default) =>
        Task.FromResult("--- a\n+++ b\n");

    public Task<IReadOnlyList<WorkItem>> ListWorkItemsAsync(IReadOnlyList<string> types, string? assignedTo, string? areaPath, bool activeOnly, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<WorkItem>>([Wi(4521)]);

    public Task<WorkItem> GetWorkItemAsync(int id, CancellationToken ct = default) =>
        Task.FromResult(Wi(id));

    public Task<IReadOnlyList<WorkItemComment>> GetWorkItemCommentsAsync(int id, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<WorkItemComment>>(
            [new WorkItemComment { ID = 1, Text = "hello", CreatedBy = User }]);

    public Task<IReadOnlyList<string>> GetWorkItemTypeStatesAsync(string workItemType, CancellationToken ct = default) =>
        Task.FromResult<IReadOnlyList<string>>(["New", "Active", "Closed"]);

    public Task UpdateWorkItemStateAsync(int id, string state, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task AddWorkItemCommentAsync(int id, string text, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }

    public Task LinkWorkItemToPRAsync(int workItemID, string prArtifactURL, CancellationToken ct = default)
    {
        Mutate();
        return Task.CompletedTask;
    }
}