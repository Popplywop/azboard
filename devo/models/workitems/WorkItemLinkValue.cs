using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>Value for linking a work item relation.</summary>
public sealed record WorkItemLinkValue
{
    [JsonPropertyName("rel")]
    public required string Rel { get; init; }

    [JsonPropertyName("url")]
    public required string URL { get; init; }

    [JsonPropertyName("attributes")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public Dictionary<string, object>? Attributes { get; init; }
}