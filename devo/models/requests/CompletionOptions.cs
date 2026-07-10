using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Specifies how a PR should be completed.</summary>
public sealed record CompletionOptions
{
    [JsonPropertyName("mergeStrategy")]
    public required string MergeStrategy { get; init; }

    [JsonPropertyName("deleteSourceBranch")]
    public bool DeleteSourceBranch { get; init; }

    [JsonPropertyName("mergeCommitMessage")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? MergeCommitMessage { get; init; }
}