from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="Worktree")


@_attrs_define
class Worktree:
    """
    Attributes:
        branch (str):
        dirty (bool):
        is_current (bool):
        is_main (bool):
        kind (str):
        locked (bool):
        path (str):
    """

    branch: str
    dirty: bool
    is_current: bool
    is_main: bool
    kind: str
    locked: bool
    path: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        branch = self.branch

        dirty = self.dirty

        is_current = self.is_current

        is_main = self.is_main

        kind = self.kind

        locked = self.locked

        path = self.path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "branch": branch,
                "dirty": dirty,
                "isCurrent": is_current,
                "isMain": is_main,
                "kind": kind,
                "locked": locked,
                "path": path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        branch = d.pop("branch")

        dirty = d.pop("dirty")

        is_current = d.pop("isCurrent")

        is_main = d.pop("isMain")

        kind = d.pop("kind")

        locked = d.pop("locked")

        path = d.pop("path")

        worktree = cls(
            branch=branch,
            dirty=dirty,
            is_current=is_current,
            is_main=is_main,
            kind=kind,
            locked=locked,
            path=path,
        )

        worktree.additional_properties = d
        return worktree

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
