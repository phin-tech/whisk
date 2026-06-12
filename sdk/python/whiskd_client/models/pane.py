from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Pane")


@_attrs_define
class Pane:
    """
    Attributes:
        id (str):
        window_id (str):
        working_dir (str):
        current_pty_id (None | str | Unset):
    """

    id: str
    window_id: str
    working_dir: str
    current_pty_id: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        window_id = self.window_id

        working_dir = self.working_dir

        current_pty_id: None | str | Unset
        if isinstance(self.current_pty_id, Unset):
            current_pty_id = UNSET
        else:
            current_pty_id = self.current_pty_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "windowId": window_id,
                "workingDir": working_dir,
            }
        )
        if current_pty_id is not UNSET:
            field_dict["currentPtyId"] = current_pty_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        window_id = d.pop("windowId")

        working_dir = d.pop("workingDir")

        def _parse_current_pty_id(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        current_pty_id = _parse_current_pty_id(d.pop("currentPtyId", UNSET))

        pane = cls(
            id=id,
            window_id=window_id,
            working_dir=working_dir,
            current_pty_id=current_pty_id,
        )

        pane.additional_properties = d
        return pane

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
