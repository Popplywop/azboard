namespace devo.models.workitems;

/// <summary>Fallback states used only if the API call to fetch states fails.</summary>
public static class WorkItemStates
{
    public static readonly IReadOnlyDictionary<string, string[]> Fallback = new Dictionary<string, string[]>
    {
        ["Bug"] = ["New", "Active", "Resolved", "Closed"],
        ["User Story"] = ["New", "Active", "Resolved", "Closed"],
        ["Task"] = ["New", "Active", "Closed"],
        ["Feature"] = ["New", "Active", "Resolved", "Closed"],
        ["Epic"] = ["New", "Active", "Resolved", "Closed"],
    };
}