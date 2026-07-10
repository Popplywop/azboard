using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>Response from a WIQL query.</summary>
public sealed record WiqlResult
{
    [JsonPropertyName("workItems")]
    public IReadOnlyList<WiqlRef> WorkItems { get; init; } = [];
}