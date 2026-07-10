using System.Text.Json.Serialization;

using devo.exceptions;
using devo.models;
using devo.models.pullrequests;
using devo.models.requests;
using devo.models.threads;

namespace devo.services;

public sealed partial class AdoClient
{
    private const int PrPageSize = 50;

    public async Task<IReadOnlyList<PullRequest>> ListPullRequestsAsync(string status, CancellationToken ct = default)
    {
        ListResponse<PullRequest> resp = await GetAsync<ListResponse<PullRequest>>(
            $"/git/pullrequests?searchCriteria.status={status}&$top={PrPageSize}", ct);
        return resp.Value;
    }

    public async Task<IReadOnlyList<PullRequest>> ListPullRequestsForRepoAsync(string repoName, string status, CancellationToken ct = default)
    {
        ListResponse<PullRequest> resp = await GetAsync<ListResponse<PullRequest>>(
            $"/git/repositories/{repoName}/pullrequests?searchCriteria.status={status}&$top={PrPageSize}", ct);
        return resp.Value;
    }

    public async Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsAsync(CancellationToken ct = default)
    {
        ListResponse<PullRequest> resp = await GetAsync<ListResponse<PullRequest>>(
            $"/git/pullrequests?searchCriteria.status=active&searchCriteria.isDraft=true&$top={PrPageSize}", ct);
        return resp.Value;
    }

    public async Task<IReadOnlyList<PullRequest>> ListDraftPullRequestsForRepoAsync(string repoName, CancellationToken ct = default)
    {
        ListResponse<PullRequest> resp = await GetAsync<ListResponse<PullRequest>>(
            $"/git/repositories/{repoName}/pullrequests?searchCriteria.status=active&searchCriteria.isDraft=true&$top={PrPageSize}", ct);
        return resp.Value;
    }

    /// <summary>Finds a PR by project-wide ID without knowing the repository.
    /// The ADO searchCriteria.pullRequestId filter is ignored by some org
    /// configurations (returns all PRs regardless), so fetch with status=all
    /// and filter client-side.</summary>
    public async Task<PullRequest> GetPullRequestByIDAsync(int prID, CancellationToken ct = default)
    {
        ListResponse<PullRequest> resp = await GetAsync<ListResponse<PullRequest>>(
            $"/git/pullrequests?searchCriteria.status=all&searchCriteria.pullRequestId={prID}", ct);
        return resp.Value.FirstOrDefault(pr => pr.PullRequestID == prID)
            ?? throw new ApiException(404, $"pull request {prID} not found");
    }

    public Task<PullRequest> GetPullRequestAsync(string repoID, int prID, CancellationToken ct = default) =>
        GetAsync<PullRequest>($"/git/repositories/{repoID}/pullrequests/{prID}", ct);

    public async Task<IReadOnlyList<CommentThread>> GetPullRequestThreadsAsync(string repoID, int prID, CancellationToken ct = default)
    {
        ListResponse<CommentThread> resp = await GetAsync<ListResponse<CommentThread>>(
            $"/git/repositories/{repoID}/pullrequests/{prID}/threads", ct);
        return resp.Value;
    }

    public Task<PullRequest> CreatePullRequestAsync(string repoID, string title, string sourceBranch, string targetBranch, string? description, bool isDraft, CancellationToken ct = default)
    {
        var body = new CreatePullRequestRequest
        {
            Title = title,
            Description = description,
            SourceRefName = sourceBranch,
            TargetRefName = targetBranch,
            IsDraft = isDraft,
        };
        return PostAsync<PullRequest>($"/git/repositories/{repoID}/pullrequests", body, ct);
    }

    /// <summary>Completes (merges) a pull request. strategy must be one of:
    /// "squash", "noFastForward", "rebase", "rebaseMerge".</summary>
    public Task MergePullRequestAsync(string repoID, int prID, string strategy, string commitMsg, bool deleteSourceBranch, CancellationToken ct = default)
    {
        var body = new CompletePullRequestRequest
        {
            Status = "completed",
            CompletionOptions = new CompletionOptions
            {
                MergeStrategy = strategy,
                DeleteSourceBranch = deleteSourceBranch,
                MergeCommitMessage = string.IsNullOrEmpty(commitMsg) ? null : commitMsg,
            },
        };
        return PatchAsync($"/git/repositories/{repoID}/pullrequests/{prID}", body, ct);
    }

    public Task AbandonPullRequestAsync(string repoID, int prID, CancellationToken ct = default) =>
        PatchAsync($"/git/repositories/{repoID}/pullrequests/{prID}",
            new StatusUpdateRequest { Status = "abandoned" }, ct);

    public Task ToggleDraftAsync(string repoID, int prID, bool isDraft, CancellationToken ct = default) =>
        PatchAsync($"/git/repositories/{repoID}/pullrequests/{prID}",
            new StatusUpdateRequest { Status = "active", IsDraft = isDraft }, ct);

    public Task CreateCommentThreadAsync(string repoID, int prID, string content, ThreadContext? threadContext = null, CancellationToken ct = default)
    {
        var body = new CreateThreadRequest
        {
            Comments = [new CreateCommentRequest { Content = content, CommentType = "text" }],
            Status = "active",
            ThreadContext = threadContext,
        };
        return PostAsync($"/git/repositories/{repoID}/pullrequests/{prID}/threads", body, ct);
    }

    public Task ReplyToCommentThreadAsync(string repoID, int prID, int threadID, string content, CancellationToken ct = default) =>
        PostAsync($"/git/repositories/{repoID}/pullrequests/{prID}/threads/{threadID}/comments",
            new CreateCommentRequest { Content = content, CommentType = "text" }, ct);

    public Task UpdateCommentThreadStatusAsync(string repoID, int prID, int threadID, string status, CancellationToken ct = default) =>
        PatchAsync($"/git/repositories/{repoID}/pullrequests/{prID}/threads/{threadID}",
            new UpdateThreadRequest { Status = status }, ct);

    public Task SetVoteAsync(string repoID, int prID, string reviewerID, int vote, CancellationToken ct = default) =>
        PutAsync($"/git/repositories/{repoID}/pullrequests/{prID}/reviewers/{reviewerID}",
            new SetVoteRequest { Vote = vote }, ct);

    /// <summary>Returns the authenticated user's ID. Tries connectionData
    /// first, falls back to the VSSPS profile endpoint.</summary>
    public async Task<string> GetCurrentUserIDAsync(CancellationToken ct = default)
    {
        try
        {
            ConnectionData data = await GetOrgAsync<ConnectionData>("/connectionData", ct);
            if (!string.IsNullOrEmpty(data.User.ID))
            {
                return data.User.ID;
            }
        }
        catch (Exception e) when (e is ApiException or HttpRequestException or System.Text.Json.JsonException)
        {
            // fall through to profile endpoint
        }

        // dev.azure.com -> vssps.dev.azure.com; x.visualstudio.com -> x.vssps.visualstudio.com
        string vsspsUrl = _orgUrl
            .Replace("dev.azure.com", "vssps.dev.azure.com")
            .Replace(".visualstudio.com", ".vssps.visualstudio.com");
        try
        {
            ProfileResponse profile = await SendAsync<ProfileResponse>(
                HttpMethod.Get, vsspsUrl + "/profile/profiles/me", body: null, ct: ct);
            if (!string.IsNullOrEmpty(profile.ID))
            {
                return profile.ID;
            }
        }
        catch (Exception e) when (e is ApiException or HttpRequestException or System.Text.Json.JsonException)
        {
            // fall through to error
        }

        throw new InvalidOperationException(
            "could not determine user ID — try setting the org URL in your config");
    }

    private sealed record ProfileResponse
    {
        [JsonPropertyName("id")]
        public required string ID { get; init; }

        [JsonPropertyName("displayName")]
        public string? DisplayName { get; init; }
    }
}