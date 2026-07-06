from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="TerminalModes")


@_attrs_define
class TerminalModes:
    """
    Attributes:
        alt_screen (bool):
        application_cursor (bool):
        bracketed_paste (bool):
        cursor_visible (bool):
        alt_screen_save (bool | Unset):
        mouse_encoding (str | Unset):
        mouse_tracking (str | Unset):
        save_cursor (bool | Unset):
    """

    alt_screen: bool
    application_cursor: bool
    bracketed_paste: bool
    cursor_visible: bool
    alt_screen_save: bool | Unset = UNSET
    mouse_encoding: str | Unset = UNSET
    mouse_tracking: str | Unset = UNSET
    save_cursor: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        alt_screen = self.alt_screen

        application_cursor = self.application_cursor

        bracketed_paste = self.bracketed_paste

        cursor_visible = self.cursor_visible

        alt_screen_save = self.alt_screen_save

        mouse_encoding = self.mouse_encoding

        mouse_tracking = self.mouse_tracking

        save_cursor = self.save_cursor

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "altScreen": alt_screen,
                "applicationCursor": application_cursor,
                "bracketedPaste": bracketed_paste,
                "cursorVisible": cursor_visible,
            }
        )
        if alt_screen_save is not UNSET:
            field_dict["altScreenSave"] = alt_screen_save
        if mouse_encoding is not UNSET:
            field_dict["mouseEncoding"] = mouse_encoding
        if mouse_tracking is not UNSET:
            field_dict["mouseTracking"] = mouse_tracking
        if save_cursor is not UNSET:
            field_dict["saveCursor"] = save_cursor

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alt_screen = d.pop("altScreen")

        application_cursor = d.pop("applicationCursor")

        bracketed_paste = d.pop("bracketedPaste")

        cursor_visible = d.pop("cursorVisible")

        alt_screen_save = d.pop("altScreenSave", UNSET)

        mouse_encoding = d.pop("mouseEncoding", UNSET)

        mouse_tracking = d.pop("mouseTracking", UNSET)

        save_cursor = d.pop("saveCursor", UNSET)

        terminal_modes = cls(
            alt_screen=alt_screen,
            application_cursor=application_cursor,
            bracketed_paste=bracketed_paste,
            cursor_visible=cursor_visible,
            alt_screen_save=alt_screen_save,
            mouse_encoding=mouse_encoding,
            mouse_tracking=mouse_tracking,
            save_cursor=save_cursor,
        )

        terminal_modes.additional_properties = d
        return terminal_modes

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
