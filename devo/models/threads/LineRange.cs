using System.Text.Json.Serialization;

namespace devo.models.threads;

/// <summary>Identifies a line/offset position in a file.</summary>
public sealed record LineRange
{
    [JsonPropertyName("line")]
    public int Line { get; init; }

    [JsonPropertyName("offset")]
    public int Offset { get; init; }
}