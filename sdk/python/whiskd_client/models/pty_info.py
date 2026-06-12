from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PTYInfo")


@_attrs_define
class PTYInfo:
    """
    Attributes:
        cols (int):
        id (str):
        origin_pane_id (str):
        origin_window_id (str):
        pane_id (str):
        rows (int):
        running (bool):
        session_id (str):
        status (str):
        window_id (str):
        working_dir (str):
        exit_code (int | None | Unset):
    """

    cols: int
    id: str
    origin_pane_id: str
    origin_window_id: str
    pane_id: str
    rows: int
    running: bool
    session_id: str
    status: str
    window_id: str
    working_dir: str
    exit_code: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cols = self.cols

        id = self.id

        origin_pane_id = self.origin_pane_id

        origin_window_id = self.origin_window_id

        pane_id = self.pane_id

        rows = self.rows

        running = self.running

        session_id = self.session_id

        status = self.status

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
                "cols": cols,
                "id": id,
                "originPaneId": origin_pane_id,
                "originWindowId": origin_window_id,
                "paneId": pane_id,
                "rows": rows,
                "running": running,
                "sessionId": session_id,
                "status": status,
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
        cols = d.pop("cols")

        id = d.pop("id")

        origin_pane_id = d.pop("originPaneId")

        origin_window_id = d.pop("originWindowId")

        pane_id = d.pop("paneId")

        rows = d.pop("rows")

        running = d.pop("running")

        session_id = d.pop("sessionId")

        status = d.pop("status")

        window_id = d.pop("windowId")

        working_dir = d.pop("workingDir")

        def _parse_exit_code(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        exit_code = _parse_exit_code(d.pop("exitCode", UNSET))

        pty_info = cls(
            cols=cols,
            id=id,
            origin_pane_id=origin_pane_id,
            origin_window_id=origin_window_id,
            pane_id=pane_id,
            rows=rows,
            running=running,
            session_id=session_id,
            status=status,
            window_id=window_id,
            working_dir=working_dir,
            exit_code=exit_code,
        )

        pty_info.additional_properties = d
        return pty_info

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
