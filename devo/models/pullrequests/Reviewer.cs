using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

public sealed record Reviewer
{
    [JsonPropertyName("displayName")]
    public required string DisplayName { get; init; }

    [JsonPropertyName("uniqueName")]
    public required string UniqueName { get; init; }

    [JsonPropertyName("id")]
    public required string ID { get; init; }

    [JsonPropertyName("vote")]
    public int Vote { get; init; }

    [JsonPropertyName("isRequired")]
    public bool IsRequired { get; init; }

    public string VoteString() => Vote switch
    {
        10 => "Approved",
        5 => "Approved with suggestions",
        0 => "No vote",
        -5 => "Waiting for author",
        -10 => "Rejected",
        _ => "Unknown"
    };

    public string VoteIcon() => Vote switch
    {
        10 => "✓",
        5 => "~",
        0 => ".",
        -5 => "⏳",
        -10 => "✗",
        _ => "?"
    };
}