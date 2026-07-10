using System.Text.Json.Serialization;

namespace devo.models.requests;

/// <summary>Body for updating a PR's status (abandon, etc.).</summary>
public sealed record StatusUpdateRequest
{
    [JsonPropertyName("status")]
    public required string Status { get; init; }

    [JsonPropertyName("isDraft")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public bool? IsDraft { get; init; }
}