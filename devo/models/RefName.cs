namespace devo.models;

internal static class RefName
{
    private const string HeadsPrefix = "refs/heads/";

    /// <summary>Returns the short branch name (strips refs/heads/).</summary>
    public static string StripPrefix(string refName) =>
        refName.StartsWith(HeadsPrefix, StringComparison.Ordinal)
            ? refName[HeadsPrefix.Length..]
            : refName;
}