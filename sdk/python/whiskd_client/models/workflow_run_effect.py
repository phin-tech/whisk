from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkflowRunEffect")


@_attrs_define
class WorkflowRunEffect:
    """
    Attributes:
        phase (str):
        preset (str):
        prompt_template_id (str):
        working_dir (str):
        auto_provision_worktree (bool | Unset):
    """

    phase: str
    preset: str
    prompt_template_id: str
    working_dir: str
    auto_provision_worktree: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        phase = self.phase

        preset = self.preset

        prompt_template_id = self.prompt_template_id

        working_dir = self.working_dir

        auto_provision_worktree = self.auto_provision_worktree

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "phase": phase,
                "preset": preset,
                "promptTemplateId": prompt_template_id,
                "workingDir": working_dir,
            }
        )
        if auto_provision_worktree is not UNSET:
            field_dict["autoProvisionWorktree"] = auto_provision_worktree

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phase = d.pop("phase")

        preset = d.pop("preset")

        prompt_template_id = d.pop("promptTemplateId")

        working_dir = d.pop("workingDir")

        auto_provision_worktree = d.pop("autoProvisionWorktree", UNSET)

        workflow_run_effect = cls(
            phase=phase,
            preset=preset,
            prompt_template_id=prompt_template_id,
            working_dir=working_dir,
            auto_provision_worktree=auto_provision_worktree,
        )

        workflow_run_effect.additional_properties = d
        return workflow_run_effect

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
