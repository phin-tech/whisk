import { execFileSync } from "node:child_process";

const input = JSON.parse(await stdin());
const issueURL = String(input.values?.url || "").trim();
const parsed = parseIssueURL(issueURL);
if (!parsed) throw new Error("issue URL must look like https://github.com/owner/repo/issues/123");
const { repo, issue } = parsed;

let title = `GitHub issue ${repo}#${issue}`;
try {
  const raw = execFileSync("gh", ["issue", "view", issue, "--repo", repo, "--json", "title"], {
    encoding: "utf8",
  });
  title = JSON.parse(raw).title || title;
} catch {
  // ponytail: gh lookup is a nicety; attachment metadata is still useful without it.
}

console.log(JSON.stringify({
  kind: "external",
  provider: "github",
  target: `${repo}#${issue}`,
  url: issueURL,
  title,
  includeInContext: true,
  meta: {
    "github/type": { type: "string", string: "issue" },
    "github/repo": { type: "string", string: repo },
    "github/number": { type: "number", number: Number(issue) },
  },
}));

function parseIssueURL(value) {
  try {
    const url = new URL(value);
    if (url.hostname !== "github.com") return null;
    const parts = url.pathname.split("/").filter(Boolean);
    if (parts.length !== 4 || parts[2] !== "issues" || !/^\d+$/.test(parts[3])) return null;
    return { repo: `${parts[0]}/${parts[1]}`, issue: parts[3] };
  } catch {
    return null;
  }
}

function stdin() {
  return new Promise((resolve, reject) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => (data += chunk));
    process.stdin.on("end", () => resolve(data));
    process.stdin.on("error", reject);
  });
}
