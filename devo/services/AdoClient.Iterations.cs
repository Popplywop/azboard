using System.Text;
using System.Text.Json.Serialization;

using devo.models;
using devo.models.pullrequests;

using DiffPlex;
using DiffPlex.Chunkers;
using DiffPlex.Model;

namespace devo.services;

public sealed partial class AdoClient
{
    public async Task<IReadOnlyList<Iteration>> GetPullRequestIterationsAsync(string repoID, int prID, CancellationToken ct = default)
    {
        ListResponse<Iteration> resp = await GetAsync<ListResponse<Iteration>>(
            $"/git/repositories/{repoID}/pullrequests/{prID}/iterations", ct);
        return resp.Value;
    }

    public async Task<IReadOnlyList<IterationChange>> GetPullRequestIterationChangesAsync(string repoID, int prID, int iterationID, CancellationToken ct = default)
    {
        // $top=2000 is the ADO maximum; without it the default page size (~100)
        // would silently truncate PRs that touch more than 100 files.
        IterationChangesResponse resp = await GetAsync<IterationChangesResponse>(
            $"/git/repositories/{repoID}/pullrequests/{prID}/iterations/{iterationID}/changes?$top=2000", ct);

        return [.. resp.ChangeEntries
            .Select(e => new IterationChange
            {
                ChangeID = e.ChangeTrackingID,
                ChangeType = e.ChangeType,
                Item = e.Item,
                OriginalPath = e.OriginalPath,
            })];
    }

    /// <summary>Fetches raw file content at a commit. $format=text returns
    /// plain bytes, which works for files of any size.</summary>
    public Task<string> GetFileContentAtCommitAsync(string repoID, string filePath, string commitID, CancellationToken ct = default)
    {
        string query = $"path={Uri.EscapeDataString(filePath)}"
            + $"&versionType=commit&version={Uri.EscapeDataString(commitID)}&$format=text";
        return GetContentAsync($"/git/repositories/{repoID}/items?{query}", ct);
    }

    /// <summary>Generates a local unified diff for a changed file.</summary>
    public async Task<string> BuildUnifiedDiffAsync(string repoID, IterationChange change, string oldCommitID, string newCommitID, CancellationToken ct = default)
    {
        string oldPath = string.IsNullOrEmpty(change.OriginalPath) ? change.Item.Path : change.OriginalPath;
        string newPath = change.Item.Path;

        string oldContent = string.Empty;
        string newContent = string.Empty;

        string changeType = change.ChangeType.ToLowerInvariant();
        if (changeType != "add")
        {
            oldContent = NormalizeLineEndings(
                await GetFileContentAtCommitAsync(repoID, oldPath, oldCommitID, ct));
        }
        if (changeType != "delete")
        {
            newContent = NormalizeLineEndings(
                await GetFileContentAtCommitAsync(repoID, newPath, newCommitID, ct));
        }

        string diff = GenerateUnifiedDiff(oldContent, newContent, "a" + oldPath, "b" + newPath);

        if (string.IsNullOrWhiteSpace(diff))
        {
            diff = $"--- a{oldPath}\n+++ b{newPath}\n(no textual changes)\n";
        }

        // For renames, prepend a git-style rename header so the viewer can tell
        // this was a rename rather than a delete+add of unrelated files.
        if (changeType == "rename" && oldPath != newPath)
        {
            diff = $"rename from {oldPath}\nrename to {newPath}\n" + diff;
        }

        return diff;
    }

    internal static string NormalizeLineEndings(string s) =>
        s.Replace("\r\n", "\n").Replace('\r', '\n');

    /// <summary>Formats DiffPlex line diffs as a unified diff with 3 lines of
    /// context, matching go-difflib output.</summary>
    internal static string GenerateUnifiedDiff(string oldText, string newText, string fromFile, string toFile)
    {
        const int context = 3;
        DiffResult diff = new Differ().CreateDiffs(
            oldText, newText, ignoreWhiteSpace: false, ignoreCase: false, new LineChunker());
        if (diff.DiffBlocks.Count == 0)
        {
            return string.Empty;
        }

        IReadOnlyList<string> a = diff.PiecesOld;
        IReadOnlyList<string> b = diff.PiecesNew;

        // group blocks whose context windows touch into one hunk
        var hunks = new List<List<DiffBlock>>();
        var current = new List<DiffBlock> { diff.DiffBlocks[0] };
        foreach (DiffBlock block in diff.DiffBlocks.Skip(1))
        {
            DiffBlock prev = current[^1];
            if (block.DeleteStartA - (prev.DeleteStartA + prev.DeleteCountA) <= context * 2)
            {
                current.Add(block);
            }
            else
            {
                hunks.Add(current);
                current = [block];
            }
        }
        hunks.Add(current);

        var sb = new StringBuilder();
        sb.Append("--- ").Append(fromFile).Append('\n');
        sb.Append("+++ ").Append(toFile).Append('\n');

        foreach (List<DiffBlock> hunk in hunks)
        {
            DiffBlock first = hunk[0];
            DiffBlock last = hunk[^1];
            int aStart = Math.Max(0, first.DeleteStartA - context);
            int aEnd = Math.Min(a.Count, last.DeleteStartA + last.DeleteCountA + context);
            int bStart = Math.Max(0, first.InsertStartB - context);
            int bEnd = Math.Min(b.Count, last.InsertStartB + last.InsertCountB + context);

            // unified format: zero-length ranges report the line before the range
            int aLine = aEnd == aStart ? aStart : aStart + 1;
            int bLine = bEnd == bStart ? bStart : bStart + 1;
            sb.Append($"@@ -{aLine},{aEnd - aStart} +{bLine},{bEnd - bStart} @@\n");

            int aPos = aStart;
            foreach (DiffBlock block in hunk)
            {
                for (; aPos < block.DeleteStartA; aPos++)
                {
                    sb.Append(' ').Append(a[aPos]).Append('\n');
                }
                for (int i = 0; i < block.DeleteCountA; i++)
                {
                    sb.Append('-').Append(a[block.DeleteStartA + i]).Append('\n');
                }
                for (int i = 0; i < block.InsertCountB; i++)
                {
                    sb.Append('+').Append(b[block.InsertStartB + i]).Append('\n');
                }
                aPos = block.DeleteStartA + block.DeleteCountA;
            }
            for (; aPos < aEnd; aPos++)
            {
                sb.Append(' ').Append(a[aPos]).Append('\n');
            }
        }

        return sb.ToString();
    }

    private sealed record IterationChangesResponse
    {
        [JsonPropertyName("changeEntries")]
        public IReadOnlyList<ChangeEntry> ChangeEntries { get; init; } = [];

        public sealed record ChangeEntry
        {
            [JsonPropertyName("changeTrackingId")]
            public int ChangeTrackingID { get; init; }

            [JsonPropertyName("changeType")]
            public required string ChangeType { get; init; }

            [JsonPropertyName("item")]
            public required ChangeItem Item { get; init; }

            [JsonPropertyName("originalPath")]
            public string? OriginalPath { get; init; }
        }
    }
}