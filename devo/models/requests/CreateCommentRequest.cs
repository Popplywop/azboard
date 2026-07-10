using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for creating a comment (in a new thread or reply).</summary>
public sealed record CreateCommentRequest
{
    [JsonPropertyName("content")]
    public required string Content { get; init; }

    [JsonPropertyName("commentType")]
    public required string CommentType { get; init; }
}