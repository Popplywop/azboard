using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

public sealed record IdentityRef
{
    [JsonPropertyName("displayName")]
    public required string DisplayName { get; init; }

    [JsonPropertyName("uniqueName")]
    public required string UniqueName { get; init; }

    [JsonPropertyName("id")]
    public required string ID { get; init; }
}