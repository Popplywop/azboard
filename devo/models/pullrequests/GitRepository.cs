using System.Text.Json.Serialization;

namespace devo.models.pullrequests;

public sealed record GitRepository
{
    [JsonPropertyName("id")]
    public required string ID { get; init; }

    [JsonPropertyName("name")]
    public required string Name { get; init; }
}