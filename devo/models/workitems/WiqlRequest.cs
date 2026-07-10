using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>Body for a WIQL query.</summary>
public sealed record WiqlRequest
{
    [JsonPropertyName("query")]
    public required string Query { get; init; }
}