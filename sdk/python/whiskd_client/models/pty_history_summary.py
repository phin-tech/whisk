from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PTYHistorySummary")


@_attrs_define
class PTYHistorySummary:
    """
    Attributes:
        created_at (datetime.datetime):
        pane_id (str):
        pty_id (str):
        session_id (str):
        window_id (str):
        working_dir (str):
        exit_code (int | None | Unset):
    """

    created_at: datetime.datetime
    pane_id: str
    pty_id: str
    session_id: str
    window_id: str
    working_dir: str
    exit_code: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        pane_id = self.pane_id

        pty_id = self.pty_id

        session_id = self.session_id

        window_id = self.window_id

        working_dir = self.working_dir

        exit_code: int | None | Unset
        if isinstance(self.exit_code, Unset):
            exit_code = UNSET
        else:
            exit_code = self.exit_code

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "paneId": pane_id,
                "ptyId": pty_id,
                "sessionId": session_id,
                "windowId": window_id,
                "workingDir": working_dir,
            }
        )
        if exit_code is not UNSET:
            field_dict["exitCode"] = exit_code

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        pane_id = d.pop("paneId")

        pty_id = d.pop("ptyId")

        session_id = d.pop("sessionId")

        window_id = d.pop("windowId")

        working_dir = d.pop("workingDir")

        def _parse_exit_code(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        exit_code = _parse_exit_code(d.pop("exitCode", UNSET))

        pty_history_summary = cls(
            created_at=created_at,
            pane_id=pane_id,
            pty_id=pty_id,
            session_id=session_id,
            window_id=window_id,
            working_dir=working_dir,
            exit_code=exit_code,
        )

        pty_history_summary.additional_properties = d
        return pty_history_summary

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
