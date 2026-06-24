from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateWorktreeRequest")


@_attrs_define
class CreateWorktreeRequest:
    """
    Attributes:
        base (str):
        branch (str):
        repo_path (str):
        override_path (str | Unset):
    """

    base: str
    branch: str
    repo_path: str
    override_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        base = self.base

        branch = self.branch

        repo_path = self.repo_path

        override_path = self.override_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "base": base,
                "branch": branch,
                "repoPath": repo_path,
            }
        )
        if override_path is not UNSET:
            field_dict["overridePath"] = override_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base = d.pop("base")

        branch = d.pop("branch")

        repo_path = d.pop("repoPath")

        override_path = d.pop("overridePath", UNSET)

        create_worktree_request = cls(
            base=base,
            branch=branch,
            repo_path=repo_path,
            override_path=override_path,
        )

        create_worktree_request.additional_properties = d
        return create_worktree_request

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
