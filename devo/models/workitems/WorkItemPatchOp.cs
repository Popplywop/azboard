using System.Text.Json.Serialization;

namespace devo.models.workitems;

/// <summary>A single operation in a JSON Patch document for work items.</summary>
public sealed record WorkItemPatchOp
{
    [JsonPropertyName("op")]
    public required string Op { get; init; }

    [JsonPropertyName("path")]
    public required string Path { get; init; }

    [JsonPropertyName("value")]
    public object? Value { get; init; }
}