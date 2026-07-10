using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>A reference to a work item returned from WIQL.</summary>
public sealed record WiqlRef
{
    [JsonPropertyName("id")]
    public int ID { get; init; }

    [JsonPropertyName("url")]
    public required string URL { get; init; }
}