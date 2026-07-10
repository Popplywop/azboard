using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

/// <summary>A changed file in an iteration.</summary>
public sealed record IterationChange
{
    [JsonPropertyName("changeId")]
    public int ChangeID { get; init; }

    [JsonPropertyName("changeType")]
    public required string ChangeType { get; init; }

    [JsonPropertyName("item")]
    public required ChangeItem Item { get; init; }

    [JsonPropertyName("originalPath")]
    public string? OriginalPath { get; init; }
}