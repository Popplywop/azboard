using System.Text.Json.Serialization;

using devo.models.threads;

namespace devo.models.requests;

/// <summary>Body for creating a new comment thread.</summary>
public sealed record CreateThreadRequest
{
    [JsonPropertyName("comments")]
    public required List<CreateCommentRequest> Comments { get; init; }

    [JsonPropertyName("status")]
    public required string Status { get; init; }

    [JsonPropertyName("threadContext")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public ThreadContext? ThreadContext { get; init; }
}