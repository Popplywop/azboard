using System.Text.Json.Serialization;

using devo.models.pullrequests;

namespace devo.models.threads;

/// <summary>A single comment within a thread.</summary>
public sealed record Comment
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("author")]
    public required IdentityRef Author { get; init; }

    [JsonPropertyName("content")]
    public string? Content { get; init; }

    [JsonPropertyName("publishedDate")]
    public DateTime PublishedDate { get; init; }

    [JsonPropertyName("commentType")]
    public string? CommentType { get; init; }

    [JsonPropertyName("isDeleted")]
    public bool IsDeleted { get; init; }
}