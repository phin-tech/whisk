from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="WorktreeBinding")


@_attrs_define
class WorktreeBinding:
    """
    Attributes:
        base (str):
        branch (str):
        created_at (datetime.datetime):
        worktree_path (str):
    """

    base: str
    branch: str
    created_at: datetime.datetime
    worktree_path: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        base = self.base

        branch = self.branch

        created_at = self.created_at.isoformat()

        worktree_path = self.worktree_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "base": base,
                "branch": branch,
                "createdAt": created_at,
                "worktreePath": worktree_path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base = d.pop("base")

        branch = d.pop("branch")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        worktree_path = d.pop("worktreePath")

        worktree_binding = cls(
            base=base,
            branch=branch,
            created_at=created_at,
            worktree_path=worktree_path,
        )

        worktree_binding.additional_properties = d
        return worktree_binding

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
