from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="Bookmark")


@_attrs_define
class Bookmark:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        kind (str):
        label (str):
        offset (int):
        pane_id (str):
        pty_id (str):
        session_id (str):
        window_id (str):
    """

    created_at: datetime.datetime
    id: str
    kind: str
    label: str
    offset: int
    pane_id: str
    pty_id: str
    session_id: str
    window_id: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        label = self.label

        offset = self.offset

        pane_id = self.pane_id

        pty_id = self.pty_id

        session_id = self.session_id

        window_id = self.window_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "label": label,
                "offset": offset,
                "paneId": pane_id,
                "ptyId": pty_id,
                "sessionId": session_id,
                "windowId": window_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        offset = d.pop("offset")

        pane_id = d.pop("paneId")

        pty_id = d.pop("ptyId")

        session_id = d.pop("sessionId")

        window_id = d.pop("windowId")

        bookmark = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            label=label,
            offset=offset,
            pane_id=pane_id,
            pty_id=pty_id,
            session_id=session_id,
            window_id=window_id,
        )

        bookmark.additional_properties = d
        return bookmark

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
