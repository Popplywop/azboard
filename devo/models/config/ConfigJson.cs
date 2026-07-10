using System.Text.Json.Serialization;

namespace devo.models.config;

public sealed class ConfigJson
{
    [JsonPropertyName("auth_method")]
    public string? AuthMethod { get; set; }

    [JsonPropertyName("org_url")]
    public string? OrgUrl { get; set; }

    [JsonPropertyName("project")]
    public string? Project { get; set; }

    [JsonPropertyName("pat")]
    public string? Pat { get; set; }

    [JsonPropertyName("repos")]
    public List<string>? Repos { get; set; }

    [JsonPropertyName("work_item_types")]
    public List<string>? WorkItemTypes { get; set; }

    [JsonPropertyName("default_merge_strategy")]
    public string? DefaultMergeStrategy { get; set; }

    [JsonPropertyName("area_path")]
    public string? AreaPath { get; set; }

    [JsonPropertyName("auto_refresh_seconds")]
    public int AutoRefreshSeconds { get; set; }
}