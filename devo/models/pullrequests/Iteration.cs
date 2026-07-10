using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

/// <summary>A PR iteration.</summary>
public sealed record Iteration
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("description")]
    public string? Description { get; init; }

    [JsonPropertyName("createdDate")]
    public DateTime CreatedDate { get; init; }

    [JsonPropertyName("sourceRefCommit")]
    public required CommitRef SourceRefCommit { get; init; }

    [JsonPropertyName("targetRefCommit")]
    public required CommitRef TargetRefCommit { get; init; }
}