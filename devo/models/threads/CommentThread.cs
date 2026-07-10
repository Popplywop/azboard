using System.Text.Json.Serialization;

namespace devo.models.threads;

/// <summary>A pull request comment thread. Named CommentThread to avoid
/// colliding with System.Threading.Thread.</summary>
public sealed record CommentThread
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("status")]
    public string? Status { get; init; }

    [JsonPropertyName("comments")]
    public IReadOnlyList<Comment> Comments { get; init; } = [];

    [JsonPropertyName("isDeleted")]
    public bool IsDeleted { get; init; }

    [JsonPropertyName("threadContext")]
    public ThreadContext? ThreadContext { get; init; }

    [JsonPropertyName("publishedDate")]
    public DateTime PublishedDate { get; init; }

    [JsonPropertyName("lastUpdatedDate")]
    public DateTime LastUpdatedDate { get; init; }
}