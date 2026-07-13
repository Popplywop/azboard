using ConsoleForge.Core;
using ConsoleForge.Layout;
using ConsoleForge.Widgets;

using devo.config;
using devo.models.pullrequests;

using devo.services;
using devo.ui.pullrequests;
using devo.ui.repopicker;

namespace devo.ui;

internal sealed record AppModel(
    IAdoClient AdoClient,
    Config Config,
    int JumpToPRID,
    string Version) : IModel, IHasSubscriptions
{
    public TabID ActiveTab { get; init; } = TabID.PullRequests;
    public Screen ActiveScreen { get; init; } = Screen.List;
    public string CurrentUserID { get; init; } = string.Empty;
    public int Width { get; init; }
    public int Height { get; init; }
    public bool ShowHelp { get; init; }
    public string? StatusMessage { get; init; }
    public PRListPage PRListPage { get; init; } = new (AdoClient, Config.Repos ?? []);
    public RepoPickerPage? Picker { get; init; }

    private TimeSpan AutoRefreshInterval => TimeSpan.FromSeconds(Config.AutoRefreshSeconds);

    #region Message Types
    private sealed record UserIDLoadedMsg(string UserID) : IMsg;
    private sealed record AppStatusClearMsg : IMsg;
    private sealed record AutoRefreshTickMsg : IMsg;
    private sealed record UpdateAvailableMsg(string Latest) : IMsg;
    private sealed record JumpToPRLoadedMsg(PullRequest PR) : IMsg;
    private sealed record JumpToPRErrorMsg(Exception Error) : IMsg;
    private sealed record OpenRepoPickerMsg : IMsg;
    private sealed record ReposSavedMsg(Exception? Error) : IMsg;
    #endregion

    public ICmd? Init()
    {
        var commands = new List<ICmd?> { PRListPage.Init(), FetchUserID() };
        if (JumpToPRID != 0)
        {
            commands.Add(FetchPRByIDCmd(JumpToPRID));
        }

        return Cmd.Batch([.. commands]);
    }

    public IReadOnlyList<(string Key, ISub Sub)> Subscriptions() =>
        Config.AutoRefreshSeconds > 0
            ? [("autorefresh", Sub.Interval(AutoRefreshInterval, _ => new AutoRefreshTickMsg()))]
            : [];

    public (IModel Model, ICmd? Cmd) Update(IMsg msg)
    {
        // resize: record dimensions, then forward to page and open modal
        if (msg is WindowResizeMsg resize)
        {
            AppModel sized = this with { Width = resize.Width, Height = resize.Height };
            (PRListPage? sizedPage, ICmd? pageCmd) = Component.Delegate(sized.PRListPage, msg);
            sized = sized with { PRListPage = sizedPage! };
            if (sized.Picker is null)
            {
                return (sized, pageCmd);
            }
            (RepoPickerPage? sizedPicker, ICmd? pickerSizeCmd) = Component.Delegate(sized.Picker, msg);
            return (sized with { Picker = sizedPicker! }, Cmd.Batch([.. new List<ICmd?> { pageCmd, pickerSizeCmd }]));
        }

        // app-level msgs first — they must not be swallowed by an open modal
        switch (msg)
        {
            case UserIDLoadedMsg m:
                return (this with { CurrentUserID = m.UserID }, null);
            case AppStatusClearMsg:
                return (this with { StatusMessage = null }, null);
            case ReposSavedMsg { Error: { } err }:
                return (this with { StatusMessage = $"Failed to save repos: {err.Message}" }, null);
            case ReposSavedMsg:
                return (this, null);
        }

        // open picker owns all remaining input until it completes
        if (Picker is not null)
        {
            (RepoPickerPage? next, ICmd? pickerCmd) = Component.Delegate(Picker, msg);
            return next is not null && Component.IsCompleted(next)
                ? ApplyPickerResult(((IComponent<PickerResult>)next).Result!, pickerCmd)
                : (this with { Picker = next! }, pickerCmd);
        }

        if (GlobalKeys.Handle(msg) is { } action)
        {
            msg = action;
        }
        switch (msg)
        {
            case QuitMsg:
                return (this, Cmd.Quit());
            case OpenRepoPickerMsg:
                RepoPickerPage picker = new(AdoClient, PRListPage.Repositories);
                if (Width > 0 && Height > 0)
                {
                    picker = picker with { Width = Width, Height = Height };
                }
                return (this with { Picker = picker }, picker.Init());
            default:
                (PRListPage? page, ICmd? cmd) = Component.Delegate(PRListPage, msg);
                return (this with { PRListPage = page! }, cmd);
        }
    }

    public IWidget View()
    {
        IWidget content = StatusMessage is null
            ? PRListPage.View()
            : new Container(Axis.Vertical, [PRListPage.View(), new TextBlock(StatusMessage)]);
        IWidget root = new BorderBox($"devo {Version} — {Config.Org}/{Config.Project}", body: content);
        return Picker is null ? root : new ZStack([root, Picker.View()]);
    }

    private static readonly KeyMap GlobalKeys = new KeyMap()
        .On(KeyBindings.Quit, () => new QuitMsg())
        .On(KeyBindings.RepoPicker, () => new OpenRepoPickerMsg());

    /// <summary>Port of app.go's RepoPickerDoneMsg/CancelMsg handling: close
    /// the modal; on confirm re-scope the PR list and refetch; on ctrl+s also
    /// persist to config and flash a status message.</summary>
    private (IModel Model, ICmd? Cmd) ApplyPickerResult(PickerResult result, ICmd? pickerCmd)
    {
        if (result.Cancelled)
        {
            return (this with { Picker = null }, pickerCmd);
        }

        PRListPage page = PRListPage with { Repositories = result.Selected, Loading = true, Error = null };
        AppModel next = this with { Picker = null, PRListPage = page };
        var commands = new List<ICmd?> { pickerCmd, page.Init() };

        if (result.Save)
        {
            next = next with { StatusMessage = $"Repos saved to config ({result.Selected.Count} selected)" };
            commands.Add(SaveReposCmd(result.Selected));
            commands.Add(Cmd.Tick(TimeSpan.FromSeconds(4), _ => new AppStatusClearMsg()));
        }

        return (next, Cmd.Batch([.. commands]));
    }

    private static ICmd SaveReposCmd(IReadOnlyList<string> repos)
        => Cmd.Run(async ct =>
        {
            try
            {
                await Config.UpdateReposAsync(repos, ct);
                return new ReposSavedMsg(null);
            }
            catch (Exception ex)
            {
                return new ReposSavedMsg(ex);
            }
        });

    #region Private Methods
    private ICmd? FetchPRByIDCmd(int jumpToPRID)
        => Cmd.Run(async ct =>
        {
            try
            {
                var pr = await AdoClient.GetPullRequestByIDAsync(jumpToPRID, ct);
                return new JumpToPRLoadedMsg(pr);
            }
            catch (Exception ex)
            {
                return new JumpToPRErrorMsg(ex);
            }
        });

    private ICmd? FetchUserID() 
        => Cmd.Run(async ct =>
        {
            try
            {
                string id = await AdoClient.GetCurrentUserIDAsync(ct);
                return new UserIDLoadedMsg(id);
            }
            catch
            {
                return new UserIDLoadedMsg(string.Empty);
            }
        });
    #endregion
}