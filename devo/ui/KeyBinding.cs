using ConsoleForge.Core;

namespace devo.ui;

internal sealed record KeyBinding(IReadOnlyList<KeyPattern> Patterns,
    string HelpKey,
    string HelpDescription);