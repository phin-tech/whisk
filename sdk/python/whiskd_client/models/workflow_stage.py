from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkflowStage")


@_attrs_define
class WorkflowStage:
    """
    Attributes:
        id (str):
        kind (str):
        name (str):
        default_prompt_template_id (str | Unset):
        default_run_preset (str | Unset):
        provision_worktree (bool | Unset):
        wip_limit (int | Unset):
    """

    id: str
    kind: str
    name: str
    default_prompt_template_id: str | Unset = UNSET
    default_run_preset: str | Unset = UNSET
    provision_worktree: bool | Unset = UNSET
    wip_limit: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind

        name = self.name

        default_prompt_template_id = self.default_prompt_template_id

        default_run_preset = self.default_run_preset

        provision_worktree = self.provision_worktree

        wip_limit = self.wip_limit

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "kind": kind,
                "name": name,
            }
        )
        if default_prompt_template_id is not UNSET:
            field_dict["defaultPromptTemplateId"] = default_prompt_template_id
        if default_run_preset is not UNSET:
            field_dict["defaultRunPreset"] = default_run_preset
        if provision_worktree is not UNSET:
            field_dict["provisionWorktree"] = provision_worktree
        if wip_limit is not UNSET:
            field_dict["wipLimit"] = wip_limit

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = d.pop("kind")

        name = d.pop("name")

        default_prompt_template_id = d.pop("defaultPromptTemplateId", UNSET)

        default_run_preset = d.pop("defaultRunPreset", UNSET)

        provision_worktree = d.pop("provisionWorktree", UNSET)

        wip_limit = d.pop("wipLimit", UNSET)

        workflow_stage = cls(
            id=id,
            kind=kind,
            name=name,
            default_prompt_template_id=default_prompt_template_id,
            default_run_preset=default_run_preset,
            provision_worktree=provision_worktree,
            wip_limit=wip_limit,
        )

        workflow_stage.additional_properties = d
        return workflow_stage

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
