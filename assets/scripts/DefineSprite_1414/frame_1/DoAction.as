if(_root.gamemode != 5)
{
   this.swapDepths(1);
   removeMovieClip(this);
   delete this.onEnterFrame;
}
swaptime = 0;
swaptotal = 2;
