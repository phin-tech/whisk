from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.session_panes_type_0 import SessionPanesType0
    from ..models.session_windows_type_0 import SessionWindowsType0


T = TypeVar("T", bound="Session")


@_attrs_define
class Session:
    """
    Attributes:
        id (str):
        name (str):
        panes (None | SessionPanesType0):
        root_dir (str):
        windows (None | SessionWindowsType0):
        project_id (str | Unset):
    """

    id: str
    name: str
    panes: None | SessionPanesType0
    root_dir: str
    windows: None | SessionWindowsType0
    project_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.session_panes_type_0 import SessionPanesType0
        from ..models.session_windows_type_0 import SessionWindowsType0

        id = self.id

        name = self.name

        panes: dict[str, Any] | None
        if isinstance(self.panes, SessionPanesType0):
            panes = self.panes.to_dict()
        else:
            panes = self.panes

        root_dir = self.root_dir

        windows: dict[str, Any] | None
        if isinstance(self.windows, SessionWindowsType0):
            windows = self.windows.to_dict()
        else:
            windows = self.windows

        project_id = self.project_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "name": name,
                "panes": panes,
                "rootDir": root_dir,
                "windows": windows,
            }
        )
        if project_id is not UNSET:
            field_dict["projectId"] = project_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.session_panes_type_0 import SessionPanesType0
        from ..models.session_windows_type_0 import SessionWindowsType0

        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        def _parse_panes(data: object) -> None | SessionPanesType0:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                panes_type_0 = SessionPanesType0.from_dict(data)

                return panes_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | SessionPanesType0, data)

        panes = _parse_panes(d.pop("panes"))

        root_dir = d.pop("rootDir")

        def _parse_windows(data: object) -> None | SessionWindowsType0:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                windows_type_0 = SessionWindowsType0.from_dict(data)

                return windows_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | SessionWindowsType0, data)

        windows = _parse_windows(d.pop("windows"))

        project_id = d.pop("projectId", UNSET)

        session = cls(
            id=id,
            name=name,
            panes=panes,
            root_dir=root_dir,
            windows=windows,
            project_id=project_id,
        )

        session.additional_properties = d
        return session

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
