from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ReadyBlockerInfo")


@_attrs_define
class ReadyBlockerInfo:
    """
    Attributes:
        id (str):
        number (int | Unset):
        run_state (str | Unset):
        stage_id (str | Unset):
        title (str | Unset):
    """

    id: str
    number: int | Unset = UNSET
    run_state: str | Unset = UNSET
    stage_id: str | Unset = UNSET
    title: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        number = self.number

        run_state = self.run_state

        stage_id = self.stage_id

        title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
            }
        )
        if number is not UNSET:
            field_dict["number"] = number
        if run_state is not UNSET:
            field_dict["runState"] = run_state
        if stage_id is not UNSET:
            field_dict["stageId"] = stage_id
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        number = d.pop("number", UNSET)

        run_state = d.pop("runState", UNSET)

        stage_id = d.pop("stageId", UNSET)

        title = d.pop("title", UNSET)

        ready_blocker_info = cls(
            id=id,
            number=number,
            run_state=run_state,
            stage_id=stage_id,
            title=title,
        )

        ready_blocker_info.additional_properties = d
        return ready_blocker_info

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
