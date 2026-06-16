export type Command = {
  id: string;
  title: string;
  shortcut?: string;
  enabled?: () => boolean;
  run: () => void | Promise<void>;
};

export type CommandItem = {
  id: string;
  title: string;
  shortcut?: string;
};

export function commandItems(commands: Command[], query = ""): CommandItem[] {
  const needle = query.trim().toLowerCase();
  return commands
    .filter((command) => command.enabled?.() !== false)
    .filter((command) => {
      if (!needle) return true;
      return command.title.toLowerCase().includes(needle) || command.id.toLowerCase().includes(needle);
    })
    .map(({ id, title, shortcut }) => (shortcut ? { id, title, shortcut } : { id, title }));
}

export async function runCommand(commands: Command[], id: string) {
  const command = commands.find((candidate) => candidate.id === id);
  if (!command || command.enabled?.() === false) return false;
  await command.run();
  return true;
}
