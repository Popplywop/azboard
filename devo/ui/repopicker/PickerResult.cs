namespace devo.ui.repopicker;

/// <summary>Outcome of the repo picker modal. Save = user pressed ctrl+s
/// (persist selection to config). Cancelled = dismissed without confirming.</summary>
internal sealed record PickerResult(
    IReadOnlyList<string> Selected,
    bool Save,
    bool Cancelled);
