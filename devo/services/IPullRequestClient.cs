using devo.models.pullrequests;
using devo.models.threads;

namespace devo.services;

public interface IPullRequestClient
{
    Task<IReadOnlyList<PullRequest>> ListPullRequestsAsync(string status, CancellationToken ct = default);
    Task<IReadOnlyList<PullRequest>> ListPullRequestsForRepoAsync(string repoName, string status, CancellationToken ct = default);
    Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsAsync(CancellationToken ct = default);
    Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsForRepoAsync(string repoName, CancellationToken ct = default);

    Task<PullRequest> GetPullRequestByIDAsync(int prID, CancellationToken ct = default);
    Task<PullRequest> GetPullRequestAsync(string repoID, int prID, CancellationToken ct = default);
    Task<IReadOnlyList<CommentThread>> GetPullRequestThreadsAsync(string repoID, int prID, CancellationToken ct = default);

    Task<PullRequest> CreatePullRequestAsync(string repoID, string title, string sourceBranch, string targetBranch, string? description, bool isDraft, CancellationToken ct = default);
    Task MergePullRequestAsync(string repoID, int prID, string strategy, string commitMsg, bool deleteSourceBranch, CancellationToken ct = default);
    Task AbandonPullRequestAsync(string repoID, int prID, CancellationToken ct = default);
    Task ToggleDraftAsync(string repoID, int prID, bool isDraft, CancellationToken ct = default);

    Task CreateCommentThreadAsync(string repoID, int prID, string content, ThreadContext? threadContext = null, CancellationToken ct = default);
    Task ReplyToCommentThreadAsync(string repoID, int prID, int threadID, string content, CancellationToken ct = default);
    Task UpdateCommentThreadStatusAsync(string repoID, int prID, int threadID, string status, CancellationToken ct = default);

    Task SetVoteAsync(string repoID, int prID, string reviewerID, int vote, CancellationToken ct = default);
    Task<string> GetCurrentUserIDAsync(CancellationToken ct = default);
}