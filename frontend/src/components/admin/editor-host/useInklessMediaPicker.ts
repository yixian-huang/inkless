import { useMemo, useRef } from "react";
import { useModalState } from "@/components/admin/editor/useModalState";
import type { ModalControls, ModalState } from "@/components/admin/editor/types-internal";
import type {
  EmbedPick,
  GalleryPick,
  MediaPickerPort,
  MediaRef,
} from "@/components/admin/editor/ports/types";

type Pending = {
  image?: (ref: MediaRef) => void;
  gallery?: (pick: GalleryPick) => void;
  video?: (ref: MediaRef) => void;
  audio?: (ref: MediaRef) => void;
  embed?: (pick: EmbedPick) => void;
};

export type MediaPickConsumers = {
  /** True if a slash/replace callback handled the pick (skip default insert). */
  consumeImagePick: (ref: MediaRef) => boolean;
  consumeGalleryPick: (pick: GalleryPick) => boolean;
  consumeVideoPick: (ref: MediaRef) => boolean;
  consumeAudioPick: (ref: MediaRef) => boolean;
  consumeEmbedPick: (pick: EmbedPick) => boolean;
};

export type InklessMediaPickerSession = {
  picker: MediaPickerPort;
  modals: ModalControls;
  state: ModalState;
  consumers: MediaPickConsumers;
};

/**
 * Per-editor media session: ModalControls for toolbar + MediaPickerPort for
 * slash/replace callbacks. Pending callbacks live in a ref so the picker
 * object stays stable for TipTap extension options.
 */
export function useInklessMediaPicker(): InklessMediaPickerSession {
  const { modals: baseModals, state } = useModalState();
  const pendingRef = useRef<Pending>({});

  const {
    setShowImagePicker,
    setShowGalleryPicker,
    setShowVideoPicker,
    setShowAudioPicker,
    setShowEmbedUrl,
  } = state;

  const picker = useMemo<MediaPickerPort>(
    () => ({
      openImage(onPick) {
        pendingRef.current.image = onPick;
        setShowImagePicker(true);
      },
      openGallery(onPick) {
        pendingRef.current.gallery = onPick;
        setShowGalleryPicker(true);
      },
      openVideo(onPick) {
        pendingRef.current.video = onPick;
        setShowVideoPicker(true);
      },
      openAudio(onPick) {
        pendingRef.current.audio = onPick;
        setShowAudioPicker(true);
      },
      openEmbed(onPick) {
        pendingRef.current.embed = onPick;
        setShowEmbedUrl(true);
      },
    }),
    [
      setShowImagePicker,
      setShowGalleryPicker,
      setShowVideoPicker,
      setShowAudioPicker,
      setShowEmbedUrl,
    ],
  );

  /** Toolbar opens without a pending callback (default insert in modals). */
  const modals = useMemo<ModalControls>(
    () => ({
      openImagePicker: () => {
        pendingRef.current.image = undefined;
        baseModals.openImagePicker();
      },
      openGalleryPicker: () => {
        pendingRef.current.gallery = undefined;
        baseModals.openGalleryPicker();
      },
      openVideoPicker: () => {
        pendingRef.current.video = undefined;
        baseModals.openVideoPicker();
      },
      openAudioPicker: () => {
        pendingRef.current.audio = undefined;
        baseModals.openAudioPicker();
      },
      openEmbedUrl: () => {
        pendingRef.current.embed = undefined;
        baseModals.openEmbedUrl();
      },
    }),
    [baseModals],
  );

  const consumers = useMemo<MediaPickConsumers>(
    () => ({
      consumeImagePick(ref) {
        const cb = pendingRef.current.image;
        pendingRef.current.image = undefined;
        if (!cb) return false;
        cb(ref);
        return true;
      },
      consumeGalleryPick(pick) {
        const cb = pendingRef.current.gallery;
        pendingRef.current.gallery = undefined;
        if (!cb) return false;
        cb(pick);
        return true;
      },
      consumeVideoPick(ref) {
        const cb = pendingRef.current.video;
        pendingRef.current.video = undefined;
        if (!cb) return false;
        cb(ref);
        return true;
      },
      consumeAudioPick(ref) {
        const cb = pendingRef.current.audio;
        pendingRef.current.audio = undefined;
        if (!cb) return false;
        cb(ref);
        return true;
      },
      consumeEmbedPick(pick) {
        const cb = pendingRef.current.embed;
        pendingRef.current.embed = undefined;
        if (!cb) return false;
        cb(pick);
        return true;
      },
    }),
    [],
  );

  return { picker, modals, state, consumers };
}
