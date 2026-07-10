using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>A single state returned by the work item type states API.</summary>
public sealed record WorkItemTypeState
{
    [JsonPropertyName("name")]
    public required string Name { get; init; }

    [JsonPropertyName("color")]
    public string? Color { get; init; }

    [JsonPropertyName("category")]
    public string? Category { get; init; }
}