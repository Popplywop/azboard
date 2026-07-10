using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

/// <summary>An Azure DevOps pull request.</summary>
public sealed record PullRequest
{
    [JsonPropertyName("pullRequestId")]
    public int PullRequestID { get; init; }

    [JsonPropertyName("title")]
    public required string Title { get; init; }

    [JsonPropertyName("description")]
    public string? Description { get; init; }

    [JsonPropertyName("status")]
    public required string Status { get; init; }

    [JsonPropertyName("creationDate")]
    public DateTime CreationDate { get; init; }

    [JsonPropertyName("closedDate")]
    public DateTime? ClosedDate { get; init; }

    [JsonPropertyName("sourceRefName")]
    public required string SourceRefName { get; init; }

    [JsonPropertyName("targetRefName")]
    public required string TargetRefName { get; init; }

    [JsonPropertyName("mergeStatus")]
    public string? MergeStatus { get; init; }

    [JsonPropertyName("isDraft")]
    public bool IsDraft { get; init; }

    [JsonPropertyName("createdBy")]
    public required IdentityRef CreatedBy { get; init; }

    [JsonPropertyName("repository")]
    public required GitRepository Repository { get; init; }

    [JsonPropertyName("reviewers")]
    public IReadOnlyList<Reviewer> Reviewers { get; init; } = [];

    /// <summary>Short source branch name (strips refs/heads/).</summary>
    [JsonIgnore]
    public string SourceBranch => RefName.StripPrefix(SourceRefName);

    /// <summary>Short target branch name (strips refs/heads/).</summary>
    [JsonIgnore]
    public string TargetBranch => RefName.StripPrefix(TargetRefName);
}