import { execFileSync } from "node:child_process";

const input = JSON.parse(await stdin());
const parsed = /^([^/]+\/[^#]+)#(\d+)$/.exec(input.target || "");
if (!parsed) {
  console.log(JSON.stringify({ delivery: "skipped", error: "expected target owner/repo#number" }));
  process.exit(0);
}

const [, repo, number] = parsed;
const raw = execFileSync("gh", [
  "issue",
  "view",
  number,
  "--repo",
  repo,
  "--json",
  "number,title,body,url,state,labels,comments",
], { encoding: "utf8" });
const issue = JSON.parse(raw);
const labels = (issue.labels || []).map((label) => label.name).join(", ");
const comments = (issue.comments || [])
  .map((comment) => `\n## Comment by ${comment.author?.login || "unknown"}\n\n${comment.body || ""}`)
  .join("\n");
const markdown = [
  `# ${repo}#${issue.number}: ${issue.title}`,
  "",
  `State: ${issue.state || ""}`,
  labels ? `Labels: ${labels}` : "",
  issue.url ? `URL: ${issue.url}` : "",
  "",
  issue.body || "",
  comments,
].filter(Boolean).join("\n");
const content = truncate(markdown, Number(input.budgetBytes || 65536));
console.log(JSON.stringify({
  title: issue.title,
  delivery: "inline",
  contentType: "text/markdown",
  content,
  sourceURL: issue.url,
}));

function truncate(value, maxBytes) {
  const encoded = Buffer.from(value);
  if (encoded.length <= maxBytes) return value;
  return encoded.subarray(0, maxBytes).toString("utf8") + "\n\n[truncated]";
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
