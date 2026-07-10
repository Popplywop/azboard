using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for setting a reviewer's vote on a PR.</summary>
public sealed record SetVoteRequest
{
    [JsonPropertyName("vote")]
    public int Vote { get; init; }
}