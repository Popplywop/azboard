using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for updating a thread's status.</summary>
public sealed record UpdateThreadRequest
{
    [JsonPropertyName("status")]
    public required string Status { get; init; }
}