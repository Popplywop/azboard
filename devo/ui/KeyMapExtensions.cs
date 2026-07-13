using ConsoleForge.Core;

namespace devo.ui;

/// <summary>Bridges KeyBinding (patterns + help metadata) onto ConsoleForge's
/// KeyMap dispatch — registers every pattern of the binding.</summary>
internal static class KeyMapExtensions
{
    public static KeyMap On(this KeyMap map, KeyBinding binding, Func<IMsg> msg)
    {
        foreach (KeyPattern pattern in binding.Patterns)
        {
            map = map.On(pattern, msg);
        }
        return map;
    }

    public static KeyMap On(this KeyMap map, KeyBinding binding, Func<KeyMsg, IMsg> msg)
    {
        foreach (KeyPattern pattern in binding.Patterns)
        {
            map = map.On(pattern, msg);
        }
        return map;
    }
}
