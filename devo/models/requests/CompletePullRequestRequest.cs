using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for completing (merging) a PR.</summary>
public sealed record CompletePullRequestRequest
{
    [JsonPropertyName("status")]
    public required string Status { get; init; }

    [JsonPropertyName("completionOptions")]
    public required CompletionOptions CompletionOptions { get; init; }
}