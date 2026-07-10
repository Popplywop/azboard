using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

/// <summary>Identifies a changed path.</summary>
public sealed record ChangeItem
{
    [JsonPropertyName("path")]
    public required string Path { get; init; }
}