using System.Text.Json.Serialization;

namespace devo.models;

/// <summary>Response from the connectionData endpoint.</summary>
public sealed record ConnectionData
{
    [JsonPropertyName("authenticatedUser")]
    public required AuthenticatedUser User { get; init; }

    public sealed record AuthenticatedUser
    {
        [JsonPropertyName("id")]
        public required string ID { get; init; }

        [JsonPropertyName("providerDisplayName")]
        public required string DisplayName { get; init; }
    }
}