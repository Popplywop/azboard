using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>An Azure DevOps work item.</summary>
public sealed record WorkItem
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("fields")]
    public required WorkItemFields Fields { get; init; }

    [JsonPropertyName("url")]
    public required string URL { get; init; }
}