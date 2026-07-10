using System.Text.Json.Serialization;

namespace devo.models.threads;

/// <summary>File path info for inline comments.</summary>
public sealed record ThreadContext
{
    [JsonPropertyName("filePath")]
    public required string FilePath { get; init; }

    [JsonPropertyName("rightFileStart")]
    public LineRange? RightFileStart { get; init; }

    [JsonPropertyName("rightFileEnd")]
    public LineRange? RightFileEnd { get; init; }
}