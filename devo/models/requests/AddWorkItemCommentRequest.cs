using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for adding a comment to a work item.</summary>
public sealed record AddWorkItemCommentRequest
{
    [JsonPropertyName("text")]
    public required string Text { get; init; }
}