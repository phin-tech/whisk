from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="RemoveWorktreeRequest")


@_attrs_define
class RemoveWorktreeRequest:
    """
    Attributes:
        also_branch (bool):
        force (bool):
        repo_path (str):
        worktree_path (str):
    """

    also_branch: bool
    force: bool
    repo_path: str
    worktree_path: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        also_branch = self.also_branch

        force = self.force

        repo_path = self.repo_path

        worktree_path = self.worktree_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "alsoBranch": also_branch,
                "force": force,
                "repoPath": repo_path,
                "worktreePath": worktree_path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        also_branch = d.pop("alsoBranch")

        force = d.pop("force")

        repo_path = d.pop("repoPath")

        worktree_path = d.pop("worktreePath")

        remove_worktree_request = cls(
            also_branch=also_branch,
            force=force,
            repo_path=repo_path,
            worktree_path=worktree_path,
        )

        remove_worktree_request.additional_properties = d
        return remove_worktree_request

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
