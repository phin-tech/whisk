from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="GateConfig")


@_attrs_define
class GateConfig:
    """
    Attributes:
        blocking (bool):
        id (str):
        kind (str):
        name (str):
        command (str | Unset):
        phase (str | Unset):
        skill (str | Unset):
    """

    blocking: bool
    id: str
    kind: str
    name: str
    command: str | Unset = UNSET
    phase: str | Unset = UNSET
    skill: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking

        id = self.id

        kind = self.kind

        name = self.name

        command = self.command

        phase = self.phase

        skill = self.skill

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "blocking": blocking,
                "id": id,
                "kind": kind,
                "name": name,
            }
        )
        if command is not UNSET:
            field_dict["command"] = command
        if phase is not UNSET:
            field_dict["phase"] = phase
        if skill is not UNSET:
            field_dict["skill"] = skill

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = d.pop("blocking")

        id = d.pop("id")

        kind = d.pop("kind")

        name = d.pop("name")

        command = d.pop("command", UNSET)

        phase = d.pop("phase", UNSET)

        skill = d.pop("skill", UNSET)

        gate_config = cls(
            blocking=blocking,
            id=id,
            kind=kind,
            name=name,
            command=command,
            phase=phase,
            skill=skill,
        )

        gate_config.additional_properties = d
        return gate_config

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
