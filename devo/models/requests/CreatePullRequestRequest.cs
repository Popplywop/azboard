using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for creating a new pull request.</summary>
public sealed record CreatePullRequestRequest
{
    [JsonPropertyName("title")]
    public required string Title { get; init; }

    [JsonPropertyName("description")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Description { get; init; }

    /// <summary>Full ref name, e.g. "refs/heads/branch-name".</summary>
    [JsonPropertyName("sourceRefName")]
    public required string SourceRefName { get; init; }

    [JsonPropertyName("targetRefName")]
    public required string TargetRefName { get; init; }

    [JsonPropertyName("isDraft")]
    public bool IsDraft { get; init; }
}