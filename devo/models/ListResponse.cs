using System.Text.Json.Serialization;

namespace devo.models;

/// <summary>Generic wrapper for Azure DevOps list API responses.</summary>
public sealed record ListResponse<T>
{
    [JsonPropertyName("count")]
    public int Count { get; init; }

    [JsonPropertyName("value")]
    public required IReadOnlyList<T> Value { get; init; }
}